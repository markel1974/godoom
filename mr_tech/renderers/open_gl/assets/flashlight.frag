#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;
in vec4 FragPosLightFlash;

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform sampler2DShadow u_flashShadowMap;
uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_flashSpaceMatrix;

uniform vec2 u_screenResolution;
uniform vec3 u_flashDir;
uniform float u_flashIntensityFactor;
uniform vec3 u_flashOffset;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;
uniform float u_flashBase;
uniform int u_enableShadows;

uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;
uniform float u_beamRatioFactor;
uniform int u_volumetricSteps;

const float PI = 3.14159265359;

float phaseHG(float cosTheta, float g) {
    float g2 = g * g;
    return (1.0 - g2) / (4.0 * PI * pow(1.0 + g2 - 2.0 * g * cosTheta, 1.5));
}

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

    vec2 duv1 = dFdx(TexCoords);
    vec2 duv2 = dFdy(TexCoords);

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

    float shadowFlash = 0.0;
    if (u_enableShadows == 1) {
        // Usa la normale geometrica già fusa dal TBN
        vec3 geoNormal = finalNormal;
        float flashBias = max(0.005 * (1.0 - clamp(dot(geoNormal, L_flash), 0.0, 1.0)), 0.001);
        shadowFlash = shadowCalculation(FragPosLightFlash, u_flashShadowMap, flashBias);
    }

    //float diffFlash = max((dot(finalNormal, L_flash) * 0.5) + u_flashBase, 0.0);
    float diffFlash = max(dot(finalNormal, L_flash) + u_flashBase, 0.0);

    float specularFlash = calculateSpecular(finalNormal, L_flash, V, isHorizontal);
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    float volFlash = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);
    for(int i = 0; i < u_volumetricSteps * 2; i++) {
        vec3 lDirFlash = normalize(flashPosView - currentPos);
        float inConeFlash = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-lDirFlash, flashSpotDir));
        if(inConeFlash > 0.01) {
            float sFlash = sampleVolumetricShadow(currentPos, u_flashSpaceMatrix, u_flashShadowMap);
            volFlash += inConeFlash * sFlash * phaseHG(dot(V, -lDirFlash), 0.5);
        }
        currentPos += rayStep;
    }

    float beamRatio = u_beamRatioFactor / float(u_volumetricSteps);
    vec3 flashBeam = vec3(0.9, 0.95, 1.0) * volFlash * u_flashIntensityFactor * beamRatio * edgeFade;

    float flashLightOcclusion = (1.0 - shadowFlash);
    vec3 litFlash = (albedo * diffFlash + vec3(specularFlash)) * (flashIntensity * 2.5) * flashLightOcclusion * vec3(1.0, 0.98, 0.9);

    FragColor = vec4(litFlash + flashBeam, 0.0);
    BrightColor = vec4(dot(litFlash + flashBeam, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? (litFlash + flashBeam) : vec3(0.0), 1.0);
}