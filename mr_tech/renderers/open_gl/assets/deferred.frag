#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 TexCoords;

uniform sampler2D u_gPositionDepth;
uniform sampler2D u_gNormal;
uniform sampler2D u_gAlbedoSpec;
uniform sampler2D u_gEmissive;
uniform sampler2D u_ssao;
uniform sampler2D u_roomShadowMap;
uniform sampler2DShadow u_flashShadowMap;

uniform vec2 u_screenResolution;
uniform float u_aoFactor;
uniform float u_ambient_light;

uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;

// Flashlight uniforms
uniform mat4 u_flashSpaceMatrix;
uniform vec3 u_flashPosView;
uniform vec3 u_flashSpotDir;
uniform float u_flashCutOff;
uniform float u_flashOuterCutOff;
uniform float u_flashIntensityFactor;

uniform mat4 u_view;

struct Light {
    vec4 pos_type;       // xyz: World position,  w: Type (0.0=Point, 1.0=Spot, 2.0=Directional, 3.0=Ambient)
    vec4 color_intensity;// xyz: RGB color,       w: Intensity
    vec4 dir_falloff;    // xyz: World direction, w: Falloff Factor
    vec4 spot_params;    // x: inner cutoff (cos), y: outer cutoff (cos), z,w: padding
};

layout(std140) uniform LightsBlock {
    Light u_lights[256];
};

uniform int u_numLights;

float calculateSpecular(vec3 normal, vec3 lightDir, vec3 viewDir, bool isHorizontal) {
    vec3 H = normalize(lightDir + viewDir);
    float NdotH = max(dot(normal, H), 0.0);
    float shininess = mix(u_shininessWall, u_shininessFloor, float(isHorizontal));
    float specBoost = mix(u_specBoostWall, u_specBoostFloor, float(isHorizontal));

    if (shininess <= 0.01) {
        return 0.0;
    }

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

const float PI = 3.14159265359;

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;

    if (fragPosLightSpace.w <= 0.0) {
        return 0.0;
    }
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));
    const int pcfCount = 2;
    for(int x = -pcfCount; x <= pcfCount; ++x) {
        for(int y = -pcfCount; y <= pcfCount; ++y) {
            shadow += texture(shadowMap, vec3(projCoords.xy + vec2(x, y) * texelSize, projCoords.z - bias));
        }
    }
    float totalSamples = pow(float(pcfCount * 2 + 1), 2.0);
    return shadow / totalSamples;
}

void main() {
    vec4 albedoSpec = texture(u_gAlbedoSpec, TexCoords);
    if(albedoSpec.a < 0.0) discard; // Empty pixel
    
    vec3 albedo = albedoSpec.rgb;
    vec4 positionDepth = texture(u_gPositionDepth, TexCoords);
    vec3 ViewPos = positionDepth.xyz;
    
    vec4 normalData = texture(u_gNormal, TexCoords);
    vec3 finalNormal = normalData.xyz;
    
    vec4 emissiveData = texture(u_gEmissive, TexCoords);
    vec3 emissive = emissiveData.rgb;
    float IsFullbright = emissiveData.a;
    
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    // 1. SSAO & AMBIENT BASE
    float ao = texture(u_ssao, TexCoords).r;
    float linearAmbient = max(pow(ao * u_aoFactor, 2.2), 0.05);
    if (IsFullbright > 0.5) linearAmbient = 1.0;
    
    // 2. ROOM DIRECTIONAL LIGHT
    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    vec3 litRoom = albedo * NdotL_room * u_ambient_light;
    
    // TODO: Room Shadows if needed
    // float shadowRoom = ...
    
    if (IsFullbright > 0.5) {
        litRoom = albedo * u_ambient_light;
    }

    vec3 baseColor = (albedo * linearAmbient) + (emissive * edgeFade) + litRoom;
    
    // 3. FLASHLIGHT
    vec3 flashLight = vec3(0.0);
    if (u_flashIntensityFactor > 0.01) {
        vec3 lightDir = normalize(u_flashPosView - ViewPos);
        float theta = dot(-lightDir, u_flashSpotDir);
        float epsilon = u_flashCutOff - u_flashOuterCutOff;
        float spotEffect = clamp((theta - u_flashOuterCutOff) / epsilon, 0.0, 1.0);
        
        if (spotEffect > 0.0) {
            float dist = length(u_flashPosView - ViewPos);
            float att = 1.0 / (1.0 + 0.005 * dist + 0.00002 * (dist * dist));
            
            float diff = max(dot(finalNormal, lightDir), 0.0);
            
            bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
            vec3 V = normalize(-ViewPos);
            float spec = calculateSpecular(finalNormal, lightDir, V, isHorizontal);
            
            // TODO: Flashlight Shadow Map
            float shadowFlash = 1.0;
            vec4 worldPos = inverse(u_view) * vec4(ViewPos, 1.0);
            vec4 fragPosLightFlash = u_flashSpaceMatrix * worldPos;
            float cosTheta = clamp(dot(finalNormal, lightDir), 0.0, 1.0);
            float bias = max(0.003 * (1.0 - cosTheta), 0.0005);
            vec3 projMain = fragPosLightFlash.xyz / fragPosLightFlash.w;
            projMain = projMain * 0.5 + 0.5;
            if(!(projMain.z > 1.0 || projMain.x < 0.0 || projMain.x > 1.0 || projMain.y < 0.0 || projMain.y > 1.0)) {
                shadowFlash = shadowCalculation(fragPosLightFlash, u_flashShadowMap, bias);
                float edgeFadeDist = smoothstep(0.0, 0.1, projMain.x) * smoothstep(1.0, 0.9, projMain.x) *
                                     smoothstep(0.0, 0.1, projMain.y) * smoothstep(1.0, 0.9, projMain.y);
                shadowFlash = mix(0.0, shadowFlash, edgeFadeDist);
            }
            
            flashLight = (albedo * diff + vec3(spec)) * att * spotEffect * u_flashIntensityFactor * shadowFlash;
        }
    }

    // 4. DYNAMIC LIGHTS
    vec3 dynamicLights = vec3(0.0);
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
    vec3 V = normalize(-ViewPos); // View Direction

    for (int i = 0; i < u_numLights; ++i) {
        int lightType = int(u_lights[i].pos_type.w);
        float intensity = max(u_lights[i].color_intensity.w, 0.0);
        if (intensity <= 0.001) continue;
        
        vec3 lightColor = u_lights[i].color_intensity.xyz;
        
        // --- OPTIMIZATION: In the future, these should be computed on CPU ---
        vec3 lightPosView = (u_view * vec4(u_lights[i].pos_type.xyz, 1.0)).xyz;
        vec3 spotDirView = normalize(mat3(u_view) * u_lights[i].dir_falloff.xyz);
        // ---------------------------------------------------------------------
        
        float falloffFactor = max(u_lights[i].dir_falloff.w, 1.0);

        vec3 L;
        float dist;
        float falloff = 1.0;
        float spotEffect = 1.0;

        if (lightType == 2) {
            L = -spotDirView;
        } else {
            vec3 diffVec = lightPosView - ViewPos;
            dist = length(diffVec);
            float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);
            if (dist > effectiveRadius) continue;
            
            L = diffVec / dist;
            falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));
            if (lightType == 1) {
                float thetaSpot = dot(-L, spotDirView);
                float cutOff = u_lights[i].spot_params.x;
                float outerCutOff = u_lights[i].spot_params.y;
                spotEffect = smoothstep(outerCutOff, cutOff, thetaSpot);
            }
        }

        float NdotL = (lightType == 3) ? mix(1.0, max(dot(finalNormal, L), 0.0), 0.5) : max(dot(finalNormal, L), 0.0);
        float specularPower = (lightType == 3) ? 0.0 : calculateSpecular(finalNormal, L, V, isHorizontal);
        
        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;
        
        dynamicLights += (diffuse + specular) * intensity * falloff * spotEffect;
    }

    vec3 finalLight = baseColor + flashLight + dynamicLights;

    FragColor = vec4(finalLight, 1.0);
    BrightColor = vec4(dot(finalLight, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalLight : vec3(0.0), 1.0);
}
