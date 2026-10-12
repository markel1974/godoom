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
uniform float u_flashFalloff;
uniform int u_enableShadows;

uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_projection;

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

const float PI = 3.14159265359;

struct ShadowLight {
    mat4 lightSpaceMatrix;
    vec4 pos_falloff;
    vec4 dir_cone;
    vec4 color_intensity;
    vec4 params;
};

layout(std140) uniform ShadowLightsBlock {
    ShadowLight u_shadowLights[8];
};

uniform sampler2DShadow u_dynamicShadowMaps[8];
uniform int u_numShadowLights;

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

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias, int volumetricSteps) {
    if (fragPosLightSpace.w <= 0.0) {
        return 0.0;
    }
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;

    if (projCoords.z > 1.0 || projCoords.z < 0.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || projCoords.y < 0.0 || projCoords.y > 1.0) {
        return 0.0;
    }

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));

    // Golden angle in radians
    const float GOLDEN_ANGLE = 2.39996323;
    float spread = 2.0;

    // Spatial noise from screen coordinates
    float noise = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) * 6.2831853;

    for(int i = 0; i < volumetricSteps; ++i) {
        float r = sqrt(float(i) + 0.5) / sqrt(float(volumetricSteps));
        float theta = float(i) * GOLDEN_ANGLE + noise;
        vec2 offset = vec2(cos(theta), sin(theta)) * r * spread;
        shadow += texture(shadowMap, vec3(projCoords.xy + offset * texelSize, currentDepth - bias));
    }
    return 1.0 - (shadow / float(volumetricSteps));
}

void main() {
    vec4 positionDepth = texture(u_gPositionDepth, TexCoords);
    if (positionDepth.z < -50000.0) {
        discard;
    }

    vec4 albedoSpec = texture(u_gAlbedoSpec, TexCoords);
    vec3 albedo = pow(albedoSpec.rgb, vec3(2.2));
    vec3 ViewPos = positionDepth.xyz;
    vec4 worldPos = u_invView * vec4(ViewPos, 1.0);
    
    vec4 normalData = texture(u_gNormal, TexCoords);
    vec3 finalNormal = normalData.xyz;
    if (dot(finalNormal, ViewPos) > 0.0) {
        finalNormal = -finalNormal;
    }
    vec3 V = normalize(-ViewPos);
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
    
    vec4 emissiveData = texture(u_gEmissive, TexCoords);
    vec3 emissive = emissiveData.rgb;
    float IsFullbright = emissiveData.a;
    
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    // 1. SSAO & AMBIENT BASE
    float ao = texture(u_ssao, TexCoords).r;
    float linearAo = max(pow(ao * u_aoFactor, 2.2), 0.05);

    // Modulate ambient by sector light level and SSAO
    vec3 ambient = albedo * (u_ambient_light * linearAo);

    // 2. ROOM DIRECTIONAL LIGHT
    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    vec3 litRoom = albedo * NdotL_room * (u_ambient_light * 0.5);

    if (IsFullbright > 0.5) {
        ambient = albedo;
        litRoom = vec3(0.0);
    }

    vec3 baseColor = ambient + litRoom + (emissive * edgeFade);
    
    // 3. FLASHLIGHT
    vec3 flashLight = vec3(0.0);
    vec3 flashBeam = vec3(0.0);
    if (u_flashIntensityFactor > 0.001) {
        vec3 flashPosView = u_flashPosView;
        vec3 flashSpotDir = normalize((u_flashSpotDir * 512.0) - flashPosView);
        vec3 L_flash = normalize(flashPosView - ViewPos);

        float flashCone = smoothstep(u_flashCutOff, u_flashOuterCutOff, dot(-L_flash, flashSpotDir));

        if (flashCone > 0.0) {
            float distToLight = length(flashPosView - ViewPos);
            float distanceFade = smoothstep(u_flashFalloff, u_flashFalloff * 0.8, distToLight);
            float flashIntensity = flashCone * (u_flashFalloff * u_flashIntensityFactor) * distanceFade;

            float diffFlash = max(dot(finalNormal, L_flash), 0.0);
            float specularFlash = (diffFlash > 0.0) ? calculateSpecular(finalNormal, L_flash, V, isHorizontal) : 0.0;

            // Flashlight Shadow Map
            float shadowFlash = 0.0;
            float edgeFadeDist = 1.0;
            if (u_enableShadows == 1) {
                vec4 fragPosLightFlash = u_flashSpaceMatrix * worldPos;
                if (fragPosLightFlash.w > 0.0) {
                    vec3 projMain = fragPosLightFlash.xyz / fragPosLightFlash.w;
                    projMain = projMain * 0.5 + 0.5;

                    if (projMain.z <= 1.0 && projMain.z >= 0.0 &&
                        projMain.x >= 0.0 && projMain.x <= 1.0 &&
                        projMain.y >= 0.0 && projMain.y <= 1.0) {

                        vec3 geoNormal = normalize(cross(dFdx(ViewPos), dFdy(ViewPos)));
                        if (geoNormal.z < 0.0) geoNormal = -geoNormal;
                        if (dot(geoNormal, ViewPos) > 0.0) geoNormal = -geoNormal;

                        float cosTheta = clamp(dot(geoNormal, L_flash), 0.0, 1.0);
                        float bias = max(0.0005 * (1.0 - cosTheta), 0.00005);
                        shadowFlash = shadowCalculation(fragPosLightFlash, u_flashShadowMap, bias, 16);
                        edgeFadeDist = smoothstep(0.0, 0.05, projMain.x) * smoothstep(1.0, 0.95, projMain.x) *
                                       smoothstep(0.0, 0.05, projMain.y) * smoothstep(1.0, 0.95, projMain.y);
                    } else {
                        edgeFadeDist = 0.0;
                    }
                } else {
                    edgeFadeDist = 0.0;
                }
            }

            float flashLightOcclusion = (1.0 - shadowFlash) * edgeFadeDist;
            const float intensityMultiplier = 2.5;
            const vec3 lightTint = vec3(1.0, 0.98, 0.9);
            flashLight = (albedo * diffFlash + vec3(specularFlash)) * (flashIntensity * intensityMultiplier) * flashLightOcclusion * lightTint;
        }

        // Volumetric flashlight beam through the air
        float volFlash = 0.0;
        const int volSteps = 16;
        vec3 rayStep = ViewPos / float(volSteps);
        float noise = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453);
        vec3 currentPos = rayStep * noise;

        mat4 viewToLight = u_flashSpaceMatrix * u_invView;
        vec4 currentShadowPos4 = viewToLight * vec4(currentPos, 1.0);
        vec4 shadowStep4 = viewToLight * vec4(rayStep, 0.0);
        const float hgConst = 0.0596831;

        for (int i = 0; i < volSteps * 2; i++) {
            vec3 lDirFlash = normalize(flashPosView - currentPos);
            float inConeFlash = smoothstep(u_flashCutOff, u_flashOuterCutOff, dot(-lDirFlash, flashSpotDir));

            if (inConeFlash > 0.01) {
                vec3 proj = currentShadowPos4.xyz / currentShadowPos4.w;
                proj = proj * 0.5 + 0.5;

                float sFlash = 1.0;
                if (u_enableShadows == 1 && currentShadowPos4.w > 0.0) {
                    if (proj.z <= 1.0 && proj.x >= 0.0 && proj.x <= 1.0 && proj.y >= 0.0 && proj.y <= 1.0) {
                        sFlash = texture(u_flashShadowMap, vec3(proj.xy, proj.z - 0.0001));
                    }
                }

                float cosThetaHG = dot(V, -lDirFlash);
                float phase = hgConst / pow(1.25 - cosThetaHG, 1.5);

                volFlash += inConeFlash * sFlash * phase;
            }

            currentPos += rayStep;
            currentShadowPos4 += shadowStep4;
        }

        const float beamRatio = 0.05 / float(volSteps);
        flashBeam = vec3(0.9, 0.95, 1.0) * volFlash * u_flashIntensityFactor * beamRatio * edgeFade;
    }

    // 4. DYNAMIC LIGHTS (Normal lights without shadows)
    vec3 dynamicLights = vec3(0.0);

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

    // 5. DYNAMIC SHADOW LIGHTS (Top 8 lights with shadows)
    vec3 shadowLightsColor = vec3(0.0);
    for (int i = 0; i < u_numShadowLights; ++i) {
        float intensity = u_shadowLights[i].color_intensity.w;
        if (intensity <= 0.001) continue;

        vec3 lightColor = u_shadowLights[i].color_intensity.xyz;
        vec3 lightPosWorld = u_shadowLights[i].pos_falloff.xyz;
        vec3 lightPosView = (u_view * vec4(lightPosWorld, 1.0)).xyz;

        vec3 spotDirWorld = u_shadowLights[i].dir_cone.xyz;
        vec3 spotDirView = normalize(mat3(u_view) * spotDirWorld);

        float falloffFactor = max(u_shadowLights[i].pos_falloff.w, 1.0);
        float cutOff = u_shadowLights[i].params.x;
        float outerCutOff = u_shadowLights[i].dir_cone.w;

        vec3 diffVec = lightPosView - ViewPos;
        float dist = length(diffVec);
        float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);

        if (dist > effectiveRadius) continue;

        vec3 L = diffVec / dist;
        float falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));

        float thetaSpot = dot(-L, spotDirView);
        float spotEffect = smoothstep(outerCutOff, cutOff, thetaSpot);
        if (spotEffect <= 0.0) continue;

        // Shadow map
        float dynShadow = 1.0;
        if (u_enableShadows == 1) {
            mat4 lightSpaceMatrix = u_shadowLights[i].lightSpaceMatrix;
            vec4 fragPosLightSpace = lightSpaceMatrix * worldPos;

            if (fragPosLightSpace.w > 0.0) {
                vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
                projCoords = projCoords * 0.5 + 0.5;

                if (projCoords.z <= 1.0 && projCoords.z >= 0.0 &&
                    projCoords.x >= 0.0 && projCoords.x <= 1.0 &&
                    projCoords.y >= 0.0 && projCoords.y <= 1.0) {

                    vec3 geoNormal = normalize(cross(dFdx(ViewPos), dFdy(ViewPos)));
                    if (geoNormal.z < 0.0) geoNormal = -geoNormal;
                    if (dot(geoNormal, ViewPos) > 0.0) geoNormal = -geoNormal;

                    float cosTheta = clamp(dot(geoNormal, L), 0.0, 1.0);
                    float bias = max(0.005 * (1.0 - cosTheta), 0.0005);
                    float shadowVal = 0.0;
                    if (i == 0) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[0], bias, 16);
                    else if (i == 1) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[1], bias, 16);
                    else if (i == 2) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[2], bias, 16);
                    else if (i == 3) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[3], bias, 16);
                    else if (i == 4) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[4], bias, 16);
                    else if (i == 5) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[5], bias, 16);
                    else if (i == 6) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[6], bias, 16);
                    else if (i == 7) shadowVal = shadowCalculation(fragPosLightSpace, u_dynamicShadowMaps[7], bias, 16);

                    float edgeFadeLight = smoothstep(0.0, 0.05, projCoords.x) * smoothstep(1.0, 0.95, projCoords.x) *
                                          smoothstep(0.0, 0.05, projCoords.y) * smoothstep(1.0, 0.95, projCoords.y);
                    dynShadow = (1.0 - shadowVal) * edgeFadeLight;
                } else {
                    dynShadow = 0.0;
                }
            } else {
                dynShadow = 0.0;
            }
        }

        float NdotL = max(dot(finalNormal, L), 0.0);
        float specularPower = (NdotL > 0.0) ? calculateSpecular(finalNormal, L, V, isHorizontal) : 0.0;

        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;

        shadowLightsColor += (diffuse + specular) * intensity * falloff * spotEffect * dynShadow;
    }

    vec3 finalLight = baseColor + flashLight + flashBeam + dynamicLights + shadowLightsColor;

    FragColor = vec4(finalLight, 1.0);
    BrightColor = vec4(dot(finalLight, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalLight : vec3(0.0), 1.0);

    vec4 clipPos = u_projection * vec4(ViewPos, 1.0);
    if (clipPos.w > 0.0) {
        gl_FragDepth = clamp((clipPos.z / clipPos.w) * 0.5 + 0.5, 0.0, 1.0);
    } else {
        gl_FragDepth = 1.0;
    }
}
