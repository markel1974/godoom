#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;
in vec4 FragPosLightFlash;

uniform sampler2DArray u_texture;
uniform sampler2DArray u_normalMap;
uniform sampler2DShadow u_flashShadowMap;
uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_flashSpaceMatrix;

uniform vec2 u_screenResolution;
uniform vec3 u_flashDir;
uniform float u_flashFalloff;
uniform float u_flashIntensityFactor;
uniform vec3 u_flashOffset;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;
uniform int u_enableShadows;

uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;
uniform float u_beamRatioFactor;
uniform int u_volumetricSteps;

const float PI = 3.14159265359;

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;
    if(projCoords.z > 1.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || projCoords.y < 0.0 || projCoords.y > 1.0) return 0.0;

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));
    //const int SAMPLES = u_volumetricSteps;
    const float GOLDEN_ANGLE = 2.39996323;
    float noise = randomNoise(gl_FragCoord.xy) * 6.2831853;
    float spread = 2.0;

    for(int i = 0; i < u_volumetricSteps; ++i) {
        float r = sqrt(float(i) + 0.5) / sqrt(float(u_volumetricSteps));
        float theta = float(i) * GOLDEN_ANGLE + noise;
        vec2 offset = vec2(cos(theta), sin(theta)) * r * spread;
        shadow += texture(shadowMap, vec3(projCoords.xy + offset * texelSize, currentDepth - bias));
    }
    return 1.0 - (shadow / float(u_volumetricSteps));
}

vec3 calculateNormal() {
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec3 geoNormal = normalize(cross(dp1, dp2));

    if (geoNormal.z < 0.0) geoNormal = -geoNormal;
    if (dot(geoNormal, ViewPos) > 0.0) geoNormal = -geoNormal;

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
    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

void main()
{
    if (u_flashIntensityFactor <= 0.01) discard;

    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    vec3 finalNormal = calculateNormal();
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
    vec3 V = normalize(-ViewPos);

    vec3 flashPosView = u_flashOffset;
    vec3 L_flash = normalize(flashPosView - ViewPos);
    vec3 flashSpotDir = normalize((u_flashDir * 512.0) - flashPosView);
    float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-L_flash, flashSpotDir));

    // Calcolo unificato della proiezione
    vec3 projMain = FragPosLightFlash.xyz / FragPosLightFlash.w;
    projMain = projMain * 0.5 + 0.5;

    float shadowFlash = 0.0;
    if (u_enableShadows == 1) {
        vec3 geoNormal = normalize(cross(dFdx(ViewPos), dFdy(ViewPos)));
        if (geoNormal.z < 0.0) geoNormal = -geoNormal;
        float cosTheta = clamp(dot(geoNormal, L_flash), 0.0, 1.0);
        float bias = clamp(0.0005 * tan(acos(cosTheta)), 0.00002, 0.0003);

        if(projMain.z > 1.0 || projMain.x < 0.0 || projMain.x > 1.0 || projMain.y < 0.0 || projMain.y > 1.0) {
            shadowFlash = 0.0;
        } else {
            shadowFlash = shadowCalculation(FragPosLightFlash, u_flashShadowMap, bias);
            float edgeFadeDist = smoothstep(0.0, 0.1, projMain.x) * smoothstep(1.0, 0.9, projMain.x) *
            smoothstep(0.0, 0.1, projMain.y) * smoothstep(1.0, 0.9, projMain.y);
            shadowFlash = mix(0.0, shadowFlash, edgeFadeDist);
        }
    }

    if (projMain.z > 0.9) {
        shadowFlash = mix(shadowFlash, 1.0, smoothstep(0.9, 1.0, projMain.z));
    }

    float diffFlash = max(dot(finalNormal, L_flash), 0.0);
    float specularFlash = calculateSpecular(finalNormal, L_flash, V, isHorizontal);
    float flashIntensity = flashCone * (u_flashFalloff * u_flashIntensityFactor);

    // --- SETUP OTTIMIZZATO PER IL VOLUMETRICO ---
    float volFlash = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);

    // Matrice pre-calcolata per mappare da View a Shadow Space
    mat4 viewToLight = u_flashSpaceMatrix * u_invView;
    // Trasformazione del punto di partenza e del vettore di avanzamento in spazio omogeneo 4D
    vec4 currentShadowPos4 = viewToLight * vec4(currentPos, 1.0);
    vec4 shadowStep4 = viewToLight * vec4(rayStep, 0.0);

    // Costante pre-calcolata per PhaseHG con g=0.5
    // (1.0 - 0.25) / (4.0 * PI) = 0.75 / 12.56637 = 0.0596831
    const float hgConst = 0.0596831;

    for(int i = 0; i < u_volumetricSteps * 2; i++) {
        vec3 lDirFlash = normalize(flashPosView - currentPos);
        float inConeFlash = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-lDirFlash, flashSpotDir));

        if(inConeFlash > 0.01) {
            // Divisione prospettica essenziale, senza moltiplicazioni di matrici
            vec3 proj = currentShadowPos4.xyz / currentShadowPos4.w;
            proj = proj * 0.5 + 0.5;

            float sFlash = 1.0;
            if(proj.z <= 1.0 && proj.x >= 0.0 && proj.x <= 1.0 && proj.y >= 0.0 && proj.y <= 1.0) {
                sFlash = texture(u_flashShadowMap, vec3(proj.xy, proj.z - 0.005));
            }

            // PhaseHG Inlinata e Ottimizzata
            float cosThetaHG = dot(V, -lDirFlash);
            float phase = hgConst / pow(1.25 - cosThetaHG, 1.5);

            volFlash += inConeFlash * sFlash * phase;
        }

        currentPos += rayStep;
        currentShadowPos4 += shadowStep4; // Avanzamento vettoriale a costo quasi zero
    }

    float beamRatio = u_beamRatioFactor / float(u_volumetricSteps);
    vec3 flashBeam = vec3(0.9, 0.95, 1.0) * volFlash * u_flashIntensityFactor * beamRatio * edgeFade;

    float flashLightOcclusion = (1.0 - shadowFlash);
    vec3 litFlash = (albedo * diffFlash + vec3(specularFlash)) * (flashIntensity * 2.5) * flashLightOcclusion * vec3(1.0, 0.98, 0.9);

    FragColor = vec4(litFlash + flashBeam, 0.0);
    BrightColor = vec4(dot(litFlash + flashBeam, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? (litFlash + flashBeam) : vec3(0.0), 1.0);
}