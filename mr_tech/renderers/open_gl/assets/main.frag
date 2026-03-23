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

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

// Filtro di Vogel per ombre morbide
float ShadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;

    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;

    // FIX FONDAMENTALE TORCIA: Se campioniamo fuori dalla Shadow Map, restituisci LUCE PIENA (0.0 = no ombra).
    // Impedisce alla torcia di auto-oscurare i suoi stessi bordi o i pavimenti lontani.
    if(projCoords.z > 1.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || projCoords.y < 0.0 || projCoords.y > 1.0) {
        return 0.0;
    }

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

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));

    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    vec2 ssaoCoords = gl_FragCoord.xy / u_screenResolution;
    float ao = texture(u_ssao, ssaoCoords).r;

    // --- VETTORI STANZA ---
    vec3 L_room_point = normalize(LightCenterView - ViewPos);
    vec3 spotDirRoom = normalize(mat3(u_view) * vec3(0.0, -1.0, 0.0));

    float cosThetaRoom = dot(-L_room_point, spotDirRoom);
    float directSpot = smoothstep(0.30, 0.50, cosThetaRoom);
    vec3 bounceDir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float cosThetaBounce = dot(-L_room_point, bounceDir);
    float bounceSpot = smoothstep(0.0, 0.80, cosThetaBounce) * 0.15;

    float roomSpotIntensity = max(directSpot, bounceSpot);

    // --- VETTORI TORCIA ---
    vec3 flashPosView = u_flashOffset;

    // Vettore L per Blinn-Phong (dalla geometria all'offset della torcia)
    vec3 L_flash = normalize(flashPosView - ViewPos);

    // FIX PARALLASSE DEFINITIVO (Y-Shear Aware):
    // Essendo u_flashDir (0, Y-shear, -1), moltiplicandolo per 512.0 otteniamo
    // l'esatto punto 3D in View-Space che si trova a profondità Z = -512.0
    // e allineato al centro focale dello schermo.
    vec3 targetPos = u_flashDir * 512.0;

    // Generiamo il vettore direzionale del fascio luminoso convergente
    vec3 flashSpotDir = normalize(targetPos - flashPosView);

    // Calcoliamo l'attenuazione radiale del cono rispetto al nuovo asse
    float cosThetaFlash = dot(-L_flash, flashSpotDir);
    float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, cosThetaFlash);

    vec3 finalNormal = NormalView;
    if (dot(finalNormal, finalNormal) < 0.01) {
        finalNormal = vec3(0.0, 1.0, 0.0);
    } else {
        finalNormal = normalize(finalNormal);
    }

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) > 0.1) {
        vec3 unpacked = (mapColor * 2.0) - 1.0;
        vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

        vec3 dp1 = dFdx(ViewPos);
        vec3 dp2 = dFdy(ViewPos);
        vec2 duv1 = dFdx(TexCoords);
        vec2 duv2 = dFdy(TexCoords);

        vec3 dp2perp = cross(dp2, finalNormal);
        vec3 dp1perp = cross(finalNormal, dp1);
        vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
        vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

        float denom = max(dot(T, T), dot(B, B));
        if (denom > 1e-5) {
            float invmax = inversesqrt(denom);
            mat3 TBN = mat3(T * invmax, B * invmax, finalNormal);
            finalNormal = normalize(TBN * mapNormal);
        }
    }

    // --- OMBRE DINAMICHE ---
    float shadowRoom = 0.0;
    float shadowFlash = 0.0;

    if (u_enableShadows == 1) {
        vec3 geoNormal = normalize(NormalView);
        float ndotlRoom = clamp(dot(geoNormal, -spotDirRoom), 0.0, 1.0);
        float roomBias = max(0.002 * (1.0 - ndotlRoom), 0.0005);

        // Aumentato il bias della torcia per prevenire il self-shadowing estremo sui pavimenti
        float ndotlFlash = clamp(dot(geoNormal, L_flash), 0.0, 1.0);
        float flashBias = max(0.005 * (1.0 - ndotlFlash), 0.001);

        shadowRoom = ShadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
        shadowFlash = ShadowCalculation(FragPosLightFlash, u_flashShadowMap, flashBias);
    }

    // --- RIFLESSI E ILLUMINAZIONE ---
    float bumpRoom = (max(dot(finalNormal, L_room_point), 0.0) * 0.2) + 1.0;
    float diffFlash = max((dot(finalNormal, L_flash) * 0.5) + u_flashBase, 0.0);

    vec3 V = normalize(-ViewPos);
    vec3 H_room = L_room_point + V;
    float NdotH_room = length(H_room) > 0.0001 ? max(dot(finalNormal, normalize(H_room)), 0.0) : 0.0;

    vec3 H_flash = L_flash + V;
    float NdotH_flash = length(H_flash) > 0.0001 ? max(dot(finalNormal, normalize(H_flash)), 0.0) : 0.0;

    float isHorizontal = step(0.8, abs(finalNormal.y));
    float shininess = mix(u_shininessWall, u_shininessFloor, isHorizontal);
    float specBoost = mix(u_specBoostWall, u_specBoostFloor, isHorizontal);

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    float specularRoom = clamp(pow(NdotH_room, shininess) * specBoost, 0.0, 1.0) * energyConservation;
    float specularFlash = clamp(pow(NdotH_flash, shininess) * specBoost, 0.0, 1.0) * energyConservation;

    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float roomFalloff = exp(-FragDepth * decayRate * 0.1);

    // Attenuazione torcia
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    // --- RAYMARCHING VOLUMETRICO ---
    float volumetricScattering = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);

    float jitter = randomNoise(gl_FragCoord.xy);
    vec3 currentPos = rayStep * jitter;

    for(int i = 0; i < u_volumetricSteps; i++) {
        vec3 toLight = LightCenterView - currentPos;
        vec3 lDir = normalize(toLight);
        float cosTheta = dot(-lDir, spotDirRoom);
        float inCone = smoothstep(0.30, 0.50, cosTheta);
        volumetricScattering += inCone;
        currentPos += rayStep;
    }

    float beamRatio = volumetricScattering / float(u_volumetricSteps);
    vec3 beamColor = vec3(1.0, 0.95, 0.85) * (beamRatio * u_beamRatioFactor) * roomFalloff * edgeFade;

    // --- MIX FINALE (IL SEGRETO DELLA TORCIA DOMINANTE) ---
    float roomLightOcclusion = (1.0 - shadowRoom);
    float flashLightOcclusion = (1.0 - shadowFlash);

    // 1. Sollevamento del Black-Point in Spazio Lineare
    // Il clamp inferiore (0.02) garantisce che le zone d'ombra non collassino allo zero assoluto (RGB 0,0,0)
    float linearAmbient = max(pow(ao * u_aoFactor, 2.2), 0.02);

    // 2. Disaccoppiamento Radianza Ambientale vs Direzionale
    // L'ambient base decade più dolcemente per preservare i dettagli nei volumi in penombra
    vec3 ambientBase = albedo * bumpRoom * linearAmbient * max(roomFalloff, 0.15);
    vec3 directRoom = (albedo * bumpRoom * roomSpotIntensity * u_roomSpotIntensityFactor + vec3(specularRoom * roomSpotIntensity)) * roomLightOcclusion * roomFalloff;

    vec3 litRoom = ambientBase + directRoom;

    vec3 flashColor = vec3(1.0, 0.98, 0.9);

    // 3. Torcia Sovrascrivente (Overdrive ricalibrato per flashFactor 3.5)
    float overdrive = 2.5;
    vec3 litFlash = (albedo * diffFlash + vec3(specularFlash)) * (flashIntensity * overdrive) * flashLightOcclusion * flashColor;

    vec3 emissive = texture(u_emissiveMap, TexCoords).rgb;

    vec3 linearColor = max(litRoom + litFlash, 0.0);
    linearColor += beamColor;

    linearColor += (emissive * u_emissiveIntensity) * edgeFade;

    FragColor = vec4(linearColor, texColor.a);
    float brightness = dot(linearColor, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(brightness > 3.0 ? linearColor : vec3(0.0), 1.0);
}