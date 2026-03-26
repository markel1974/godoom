#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;
in vec3 ViewPos;
in vec3 LightCenterView;
in vec3 NormalView;

in vec4 FragPosLightRoom;
in vec4 FragPosLightFlash;

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform sampler2D u_ssao;
uniform sampler2DShadow u_roomShadowMap;
uniform sampler2DShadow u_flashShadowMap;
uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

uniform vec2 u_screenResolution;
uniform float u_ambient_light;
uniform vec3 u_flashDir;
uniform float u_flashIntensityFactor;
uniform vec3 u_flashOffset;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;
uniform float u_flashBase;
uniform int u_enableShadows;
uniform sampler2D u_emissiveMap;

uniform float u_emissiveIntensity;
uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;
uniform float u_beamRatioFactor;
uniform float u_aoFactor;
uniform float u_roomSpotIntensityFactor;
uniform int u_volumetricSteps;

const float PI = 3.14159265359;

// ==========================================================
// CORE MATH & UTILS
// ==========================================================
float phaseHG(float cosTheta, float g) {
    float g2 = g * g;
    return (1.0 - g2) / (4.0 * PI * pow(1.0 + g2 - 2.0 * g * cosTheta, 1.5));
}

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

// ==========================================================
// SHADOW MAPPING
// ==========================================================
float sampleVolumetricShadow(vec3 posView, mat4 lightSpaceMatrix, sampler2DShadow shadowMap) {
    vec4 worldPos = u_invView * vec4(posView, 1.0);
    vec4 shadowPos = lightSpaceMatrix * worldPos;
    vec3 proj = shadowPos.xyz / shadowPos.w;
    proj = proj * 0.5 + 0.5;
    if(proj.z > 1.0 || proj.x < 0.0 || proj.x > 1.0 || proj.y < 0.0 || proj.y > 1.0) return 1.0;
    return texture(shadowMap, vec3(proj.xy, proj.z - 0.005));
}

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;
    if(projCoords.z > 1.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || projCoords.y < 0.0 || projCoords.y > 1.0) return 0.0;

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));
    const int SAMPLES = 16;
    const float GOLDEN_ANGLE = 2.39996323;
    float noise = randomNoise(gl_FragCoord.xy) * 6.2831853;
    float spread = 2.0;

    for(int i = 0; i < SAMPLES; ++i) {
        float r = sqrt(float(i) + 0.5) / sqrt(float(SAMPLES));
        float theta = float(i) * GOLDEN_ANGLE + noise;
        vec2 offset = vec2(cos(theta), sin(theta)) * r * spread;
        shadow += texture(shadowMap, vec3(projCoords.xy + offset * texelSize, currentDepth - bias));
    }
    return 1.0 - (shadow / float(SAMPLES));
}

// ==========================================================
// GEOMETRY & PBR
// ==========================================================
vec3 calculateNormal(vec3 baseNormal) {
    if (dot(baseNormal, baseNormal) < 0.01) return vec3(0.0, 1.0, 0.0);
    vec3 normal = normalize(baseNormal);

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) < 0.1) return normal;

    vec3 unpacked = (mapColor * 2.0) - 1.0;
    vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec2 duv1 = dFdx(TexCoords);
    vec2 duv2 = dFdy(TexCoords);

    vec3 dp2perp = cross(dp2, normal);
    vec3 dp1perp = cross(normal, dp1);
    vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
    vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

    float denom = max(dot(T, T), dot(B, B));
    if (denom > 1e-5) {
        float invmax = inversesqrt(denom);
        mat3 TBN = mat3(T * invmax, B * invmax, normal);
        return normalize(TBN * mapNormal);
    }
    return normal;
}

float calculateSpecular(vec3 normal, vec3 lightDir, vec3 viewDir, bool isHorizontal) {
    vec3 H = normalize(lightDir + viewDir);
    float NdotH = max(dot(normal, H), 0.0);

    float shininess = mix(u_shininessWall, u_shininessFloor, float(isHorizontal));
    float specBoost = mix(u_specBoostWall, u_specBoostFloor, float(isHorizontal));

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

// ==========================================================
// VOLUMETRIC RAYMARCHING
// ==========================================================
vec3 calculateVolumetric(vec3 viewDir, vec3 flashPosView, vec3 flashSpotDir, vec3 spotDirRoom, float edgeFade, float distanceFalloff) {
    float volRoom = 0.0;
    float volFlash = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);

    bool isFlashOn = u_flashIntensityFactor > 0.01;

    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float normalizedIntensity = clamp(1.0 - (decayRate / 5.0), 0.0, 1.0);

    for(int i = 0; i < u_volumetricSteps * 2; i++) {
        // 1. ROOM: Nebbia ambientale diffusa
        // FIX: Usiamo direttamente LightCenterView che è già nello spazio corretto
        float distToCenter = length(LightCenterView - currentPos);
        float fogGlow = exp(-distToCenter * 0.005) * normalizedIntensity;

        float sRoom = sampleVolumetricShadow(currentPos, u_roomSpaceMatrix, u_roomShadowMap);
        volRoom += fogGlow * sRoom * 0.15;

        // 2. FLASH: Cono volumetrico
        if(isFlashOn) {
            vec3 lDirFlash = normalize(flashPosView - currentPos);
            float inConeFlash = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-lDirFlash, flashSpotDir));
            if(inConeFlash > 0.01) {
                float sFlash = sampleVolumetricShadow(currentPos, u_flashSpaceMatrix, u_flashShadowMap);
                volFlash += inConeFlash * sFlash * phaseHG(dot(viewDir, -lDirFlash), 0.5);
            }
        }
        currentPos += rayStep;
    }

    float beamRatio = u_beamRatioFactor / float(u_volumetricSteps);

    vec3 roomBeam = vec3(1.0, 0.95, 0.85) * volRoom;
    vec3 flashBeam = vec3(0.9, 0.95, 1.0) * volFlash * u_flashIntensityFactor;

    return (roomBeam + flashBeam) * beamRatio * distanceFalloff * edgeFade;
}

// ==========================================================
// MAIN PIPELINE
// ==========================================================
void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);
    float ao = texture(u_ssao, screenUV).r;

    // ==========================================================
    // SETUP VETTORIALE
    // ==========================================================
    vec3 V = normalize(-ViewPos);

    // Vettore direzionale fisso verso l'alto convertito in View Space
    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    vec3 spotDirRoom = L_room_dir; // Allineato al calcolo direzionale

    vec3 flashPosView = u_flashOffset;
    vec3 L_flash = normalize(flashPosView - ViewPos);
    vec3 flashSpotDir = normalize((u_flashDir * 512.0) - flashPosView);
    float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-L_flash, flashSpotDir));

    vec3 finalNormal = calculateNormal(NormalView);
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;

    // Ombre
    float shadowRoom = 0.0;
    float shadowFlash = 0.0;
    if (u_enableShadows == 1) {
        vec3 geoNormal = normalize(NormalView);
        // Riduciamo l'ordine di grandezza, il Normal Offset farà il lavoro pesante
        float roomBias = max(0.0005 * (1.0 - clamp(dot(geoNormal, spotDirRoom), 0.0, 1.0)), 0.0001);
        float flashBias = max(0.005 * (1.0 - clamp(dot(geoNormal, L_flash), 0.0, 1.0)), 0.001);

        shadowRoom = shadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
        shadowFlash = shadowCalculation(FragPosLightFlash, u_flashShadowMap, flashBias);
    }

    // Materiali e Speculari accoppiati
    float bumpRoom = (max(dot(finalNormal, L_room_dir), 0.0) * 0.2) + 1.0;
    float diffFlash = max((dot(finalNormal, L_flash) * 0.5) + u_flashBase, 0.0);

    //TODO bug e' sbagliato il calcolo della luce speculare nella stanza
    float specularRoom = 0.0;//calculateSpecular(finalNormal, L_room_dir, V, isHorizontal);
    float specularFlash = calculateSpecular(finalNormal, L_flash, V, isHorizontal);

    // Illuminazione HDR Sector
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float normalizedIntensity = clamp(1.0 - (decayRate / 5.0), 0.0, 1.0);
    float sectorHdrFactor = 100.0;
    float sectorLightLevel = pow(normalizedIntensity, 2.2) * sectorHdrFactor;

    // Falloff forzato a 1.0 per debug
    //float roomFalloff = 1.0;
    float roomFalloff = exp(-FragDepth * decayRate * 0.015);
    float finalSectorLight = sectorLightLevel * roomFalloff;


    // Intensità Flashlight
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    // Volumetria
    vec3 beamColor = calculateVolumetric(V, flashPosView, flashSpotDir, spotDirRoom, edgeFade, roomFalloff);

    // Composizione Finale
    float roomLightOcclusion = (1.0 - shadowRoom);
    float flashLightOcclusion = (1.0 - shadowFlash);
    float linearAmbient = max(pow(ao * u_aoFactor, 2.2), 0.05);
    float shadowFactor = (0.3 + 0.7 * roomLightOcclusion);

    vec3 litRoom = albedo * bumpRoom * linearAmbient * max(finalSectorLight * shadowFactor, 0.01);
    litRoom += vec3(specularRoom) * finalSectorLight * roomLightOcclusion * u_roomSpotIntensityFactor;

    vec3 litFlash = (albedo * diffFlash + vec3(specularFlash)) * (flashIntensity * 2.5) * flashLightOcclusion * vec3(1.0, 0.98, 0.9);
    vec3 emissive = texture(u_emissiveMap, TexCoords).rgb;

    vec3 linearColor = max(litRoom + litFlash, 0.0) + beamColor + (emissive * u_emissiveIntensity) * edgeFade;

    FragColor = vec4(linearColor, texColor.a);
    BrightColor = vec4(dot(linearColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? linearColor : vec3(0.0), 1.0);
}