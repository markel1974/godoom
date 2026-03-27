#version 330 core
out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D gAlbedoSpec;
uniform sampler2D gNormalEmiss;
uniform sampler2D gPositionDepth;
uniform sampler2DShadow u_flashShadowMap;

uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_flashSpaceMatrix;

uniform vec3 u_flashDir;
uniform float u_flashIntensityFactor;
uniform vec3 u_flashOffset;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;
uniform float u_flashBase;
uniform int u_volumetricSteps;

const float PI = 3.14159265359;
const float GOLDEN_ANGLE = 2.39996323;

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
    const int SAMPLES = 16;
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

float phaseHG(float cosTheta, float g) {
    float g2 = g * g;
    return (1.0 - g2) / (4.0 * PI * pow(1.0 + g2 - 2.0 * g * cosTheta, 1.5));
}

float sampleVolumetricShadow(vec3 posView, mat4 lightSpaceMatrix, sampler2DShadow shadowMap) {
    vec4 worldPos = u_invView * vec4(posView, 1.0);
    vec4 shadowPos = lightSpaceMatrix * worldPos;
    vec3 proj = shadowPos.xyz / shadowPos.w;
    proj = proj * 0.5 + 0.5;
    if(proj.z > 1.0 || proj.x < 0.0 || proj.x > 1.0 || proj.y < 0.0 || proj.y > 1.0) return 1.0;
    return texture(shadowMap, vec3(proj.xy, proj.z - 0.005));
}

void main() {
    if (u_flashIntensityFactor < 0.01) discard;

    vec4 posDepth = texture(gPositionDepth, TexCoords);
    if (posDepth.w == 0.0) discard; // Skybox

    vec3 albedo = texture(gAlbedoSpec, TexCoords).rgb;
    vec3 normal = normalize(texture(gNormalEmiss, TexCoords).rgb);
    vec3 viewPos = posDepth.xyz;
    float fragDepth = posDepth.w;

    vec3 V = normalize(-viewPos);
    vec3 flashPosView = u_flashOffset;
    vec3 L_flash = flashPosView - viewPos;
    float dist = length(L_flash);
    L_flash = L_flash / dist;

    vec3 flashSpotDir = normalize((u_flashDir * 512.0) - flashPosView);
    float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-L_flash, flashSpotDir));

    vec4 worldPos = u_invView * vec4(viewPos, 1.0);
    vec4 fragPosLightFlash = u_flashSpaceMatrix * worldPos;

    float flashBias = max(0.005 * (1.0 - clamp(dot(normal, L_flash), 0.0, 1.0)), 0.001);
    float shadowFlash = shadowCalculation(fragPosLightFlash, u_flashShadowMap, flashBias);

    float diffFlash = max((dot(normal, L_flash) * 0.5) + u_flashBase, 0.0);
    float flashFalloff = 1.0 / (1.0 + (0.05 * fragDepth) + 0.005 * (fragDepth * fragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    vec3 litFlash = albedo * diffFlash * (flashIntensity * 2.5) * (1.0 - shadowFlash) * vec3(1.0, 0.98, 0.9);

    // Volumetria Torcia
    float volFlash = 0.0;
    vec3 rayStep = viewPos / float(u_volumetricSteps);
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

    vec3 flashBeam = vec3(0.9, 0.95, 1.0) * volFlash * u_flashIntensityFactor * (0.05 / float(u_volumetricSteps));

    FragColor = vec4(litFlash + flashBeam, 1.0);
}