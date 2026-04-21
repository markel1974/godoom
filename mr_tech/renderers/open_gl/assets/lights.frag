#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;
in vec4 FragPosLightRoom;

uniform sampler2DArray u_texture;
uniform sampler2DArray u_normalMap;
uniform sampler2DShadow u_roomShadowMap;
uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_roomSpaceMatrix;

uniform vec2 u_screenResolution;
uniform float u_ambient_light;
uniform int u_enableShadows;
uniform int u_volumetricSteps;
uniform float u_beamRatioFactor;
uniform int u_numLights;

uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;

const float PI = 3.14159265359;

// --- NUOVO UBO PER LUCI MULTIPLE (Allineamento std140 rigoroso a 16 byte) ---
struct Light {
    vec4 pos_type;       // xyz: Posizione World, w: Tipo (0.0=Point, 1.0=Spot, 2.0=Directional)
    vec4 color_intensity;// xyz: Colore RGB,      w: Intensità
    vec4 dir_falloff;    // xyz: Direzione World, w: Falloff Factor
    vec4 spot_params;    // x: inner cutoff (cos), y: outer cutoff (cos), z,w: padding
};

layout(std140) uniform LightsBlock {
    Light u_lights[256];
};

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

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

vec3 calculateNormal() {
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec3 geoNormal = normalize(cross(dp1, dp2));

    // Sicurezza Anti-Backface antisfarfallio
    if (geoNormal.z < 0.0) {
        geoNormal = -geoNormal;
    }

    // Sicurezza Anti-Backface: forza la normale a guardare la camera
    if (dot(geoNormal, ViewPos) > 0.0) {
        geoNormal = -geoNormal;
    }

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) < 0.1) return geoNormal;

    vec3 unpacked = (mapColor * 2.0) - 1.0;
    vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

    vec2 duv1 = dFdx(TexCoords.xy);
    vec2 duv2 = dFdy(TexCoords.xy);

    vec3 dp2perp = cross(dp2, geoNormal);
    vec3 dp1perp = cross(geoNormal, dp1);
    vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
    vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

    float denom = max(dot(T, T), dot(B, B));
    if (denom > 1e-5 && denom < 1e6) {
        float invmax = inversesqrt(denom);
        mat3 TBN = mat3(T * invmax, B * invmax, geoNormal);
        return normalize(TBN * mapNormal);
    }
    return geoNormal;
}

float calculateSpecular(vec3 normal, vec3 lightDir, vec3 viewDir, bool isHorizontal) {
    vec3 H = normalize(lightDir + viewDir);
    float NdotH = max(dot(normal, H), 0.0);
    float shininess = mix(u_shininessWall, u_shininessFloor, float(isHorizontal));
    float specBoost = mix(u_specBoostWall, u_specBoostFloor, float(isHorizontal));

    if (shininess <= 0.01) return 0.0;

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    vec3 finalNormal = calculateNormal();

    // VARIABILI NECESSARIE PER LO SPECULARE
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
    vec3 V = normalize(-ViewPos); // View Direction

    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));

    // OMBRE STANZA
    float shadowRoom = 0.0;
    if (u_enableShadows == 1) {
        // Usa la normale geometrica già fusa dal TBN
        vec3 geoNormal = finalNormal;
        float roomBias = max(0.05 * (1.0 - clamp(dot(geoNormal, L_room_dir), 0.0, 1.0)), 0.005);
        shadowRoom = shadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
    }
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    float shadowFactor = 1.0 - shadowRoom;
    vec3 litRoom = albedo * NdotL_room * u_ambient_light * shadowFactor;
    // NEBBIA VOLUMETRICA
    float volRoom = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);
    for(int i = 0; i < u_volumetricSteps * 2; i++) {
        float fogGlow = exp(-length(currentPos) * 0.005) * u_ambient_light;
        float sRoom = sampleVolumetricShadow(currentPos, u_roomSpaceMatrix, u_roomShadowMap);
        volRoom += fogGlow * sRoom * 0.15;
        currentPos += rayStep;
    }
    vec3 roomBeam = vec3(1.0, 0.95, 0.85) * volRoom * (u_beamRatioFactor / float(u_volumetricSteps)) * edgeFade;

    // LUCI DINAMICHE
    vec3 dynamicLights = vec3(0.0);

    for (int i = 0; i < u_numLights; ++i) {
        int lightType = int(u_lights[i].pos_type.w);
        float intensity = max(u_lights[i].color_intensity.w, 0.0);

        // Se la luce è spenta, salta
        if (intensity <= 0.001) continue;

        vec3 lightColor = u_lights[i].color_intensity.xyz;
        vec3 lightPosView = (u_view * vec4(u_lights[i].pos_type.xyz, 1.0)).xyz;
        vec3 spotDirView = normalize(mat3(u_view) * u_lights[i].dir_falloff.xyz);

        // Trattiamo falloffFactor come il raggio d'azione massimo della luce
        float falloffFactor = max(u_lights[i].dir_falloff.w, 1.0);

        vec3 L;
        float dist;
        float falloff = 1.0;
        float spotEffect = 1.0;

        if (lightType == 2) {
            // --- LUCE DIREZIONALE ---
            L = -spotDirView;
        } else {
            // --- POINT & SPOT ---
            vec3 diff = lightPosView - ViewPos;
            dist = length(diff);

            // EARLY-OUT: Culling per-pixel. Se siamo oltre il raggio, la luce non ha effetto.
            // Risparmia tonnellate di calcoli su mappe grandi.
            //if (dist > falloffFactor) continue;
            //L = diff / dist;
            // Reintegro del tuo falloff esponenziale, ma ora è delimitato dal raggio
            //falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));

            float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);
            // EARLY-OUT: Ora scartiamo i pixel solo se sono VERAMENTE fuori portata
            if (dist > effectiveRadius) continue;
            L = diff / dist;
            // Il tuo decadimento esponenziale HDR originale
            falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));

            if (lightType == 1) {
                // --- LIMITATORE CONO SPOT ---
                float theta = dot(-L, spotDirView);
                float cutOff = u_lights[i].spot_params.x;
                float outerCutOff = u_lights[i].spot_params.y;
                spotEffect = smoothstep(outerCutOff, cutOff, theta);
            }
        }

        // CALCOLO DIFFUSIONE E SPECULARE
        float NdotL = max(dot(finalNormal, L), 0.0);
        float specularPower = calculateSpecular(finalNormal, L, V, isHorizontal);
        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;

        // ACCUMULO FINALE
        dynamicLights += (diffuse + specular) * intensity * falloff * spotEffect;
    }

    vec3 finalLight = litRoom + roomBeam + dynamicLights;
    FragColor = vec4(finalLight, 1.0); // Forzato a 1.0 per sicurezza sul frame buffer
    BrightColor = vec4(dot(finalLight, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalLight : vec3(0.0), 1.0);
}