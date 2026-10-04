#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec4 FragPosLightRoom;
in vec4 FragPosLightFlash;
in float IsFullbright;

uniform sampler2DArray u_texture[4];
uniform sampler2DArray u_normalMap[4];

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
uniform int u_debugLights;

const float PI = 3.14159265359;

// UBO for multiple lights (Strict 16-byte std140 alignment)
struct Light {
    vec4 pos_type;       // xyz: World position,  w: Type (0.0=Point, 1.0=Spot, 2.0=Directional)
    vec4 color_intensity;// xyz: RGB color,       w: Intensity
    vec4 dir_falloff;    // xyz: World direction, w: Falloff Factor
    vec4 spot_params;    // x: inner cutoff (cos), y: outer cutoff (cos), z,w: padding
};

layout(std140) uniform LightsBlock {
    Light u_lights[256];
};

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) {
        return texture(u_texture[0], vec3(tc.xy, l));
    }
    if (b == 1) {
        return texture(u_texture[1], vec3(tc.xy, l));
    }
    if (b == 2) {
        return texture(u_texture[2], vec3(tc.xy, l));
    }
    return texture(u_texture[3], vec3(tc.xy, l));
}

vec3 getNormal(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) {
        return texture(u_normalMap[0], vec3(tc.xy, l)).rgb;
    }
    if (b == 1) {
        return texture(u_normalMap[1], vec3(tc.xy, l)).rgb;
    }
    if (b == 2) {
        return texture(u_normalMap[2], vec3(tc.xy, l)).rgb;
    }
    return texture(u_normalMap[3], vec3(tc.xy, l)).rgb;
}

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

float sampleVolumetricShadow(vec3 posView, mat4 lightSpaceMatrix, sampler2DShadow shadowMap) {
    vec4 worldPos = u_invView * vec4(posView, 1.0);
    vec4 shadowPos = lightSpaceMatrix * worldPos;
    vec3 proj = shadowPos.xyz / shadowPos.w;
    proj = proj * 0.5 + 0.5;
    if(proj.z > 1.0 || proj.x < 0.0 || proj.x > 1.0 || proj.y < 0.0 || proj.y > 1.0) {
        return 1.0;
    }
    return texture(shadowMap, vec3(proj.xy, proj.z - 0.005));
}

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;
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

vec3 calculateNormal() {
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec3 geoNormal = normalize(cross(dp1, dp2));
    // Anti-backface flicker safety
    if (geoNormal.z < 0.0) {
        geoNormal = -geoNormal;
    }
    // Anti-backface safety: force normal to face the camera
    if (dot(geoNormal, ViewPos) > 0.0) {
        geoNormal = -geoNormal;
    }

    vec3 mapColor = getNormal(TexCoords);
    if (length(mapColor) < 0.1) {
        return geoNormal;
    }
    vec3 unpacked = (mapColor * 2.0) - 1.0;
    vec3 mapNormal = normalize(unpacked);

    vec2 duv1 = dFdx(TexCoords.xy);
    vec2 duv2 = dFdy(TexCoords.xy);

    vec3 dp2perp = cross(dp2, geoNormal);
    vec3 dp1perp = cross(geoNormal, dp1);
    vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
    vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

    float lenT = length(T);
    float lenB = length(B);
    if (lenT > 1e-8 && lenB > 1e-8) {
        mat3 TBN = mat3(T / lenT, B / lenB, geoNormal);
        return normalize(TBN * mapNormal);
    }
    return geoNormal;
}

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

void main() {
    vec4 texColor = getDiffuse(TexCoords);
    if(texColor.a < 0.5) {
        discard;
    }
    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    vec3 finalNormal = calculateNormal();

    // Specular variables
    bool isHorizontal = step(0.8, abs(finalNormal.y)) > 0.5;
    vec3 V = normalize(-ViewPos); // View Direction

    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    vec3 litRoom = albedo * NdotL_room * u_ambient_light;
    vec3 roomBeam = vec3(0.0); // By default, no volumetric fog

    //Shadow lights are in flashlights.frag
    //we ha to remove u_enableShadows, u_volumetricSteps, u_roomShadowMap
    int enableShadows = u_enableShadows;
    enableShadows = 0;

    if (enableShadows == 1) {
        // Rom shadows
        // Use the geometric normal already blended by TBN
        vec3 geoNormal = finalNormal;
        float roomBias = max(0.05 * (1.0 - clamp(dot(geoNormal, L_room_dir), 0.0, 1.0)), 0.005);
        float shadowRoom = shadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
        float shadowFactor = 1.0 - shadowRoom;
        if (IsFullbright > 0.5) {
            shadowFactor = 1.0;
            litRoom = albedo * u_ambient_light;
        } else {
            litRoom = albedo * NdotL_room * u_ambient_light * shadowFactor;
        }
        // Volumetric fog
        float volRoom = 0.0;
        if (u_volumetricSteps > 0) {
            vec3 rayStep = ViewPos / float(u_volumetricSteps);
            vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);
            for (int i = 0; i < u_volumetricSteps * 2; i++) {
                float fogGlow = exp(-length(currentPos) * 0.005) * u_ambient_light;
                float sRoom = sampleVolumetricShadow(currentPos, u_roomSpaceMatrix, u_roomShadowMap);
                volRoom += fogGlow * sRoom * 0.15;
                currentPos += rayStep;
            }
        }
        float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);
        // Calculate roomBeam ONLY if raymarching was performed
        roomBeam = vec3(1.0, 0.95, 0.85) * volRoom * (u_beamRatioFactor / float(u_volumetricSteps)) * edgeFade;
    }

    vec3 dynamicLights = vec3(0.0);

    for (int i = 0; i < u_numLights; ++i) {
        int lightType = int(u_lights[i].pos_type.w);
        float intensity = max(u_lights[i].color_intensity.w, 0.0);
        // Skip if light is disabled
        if (intensity <= 0.001) {
            continue;
        }
        vec3 lightColor = u_lights[i].color_intensity.xyz;
        vec3 lightPosView = (u_view * vec4(u_lights[i].pos_type.xyz, 1.0)).xyz;
        vec3 spotDirView = normalize(mat3(u_view) * u_lights[i].dir_falloff.xyz);
        // Treat falloff-Factor as the maximum light radius
        float falloffFactor = max(u_lights[i].dir_falloff.w, 1.0);

        vec3 L;
        float dist;
        float falloff = 1.0;
        float spotEffect = 1.0;

        if (lightType == 2) {
            // Direction light
            L = -spotDirView;
        } else {
            // Point & Spot
            vec3 diff = lightPosView - ViewPos;
            dist = length(diff);
            float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);
            // Discard pixels only if truly out of range
            if (dist > effectiveRadius) {
                continue;
            }
            L = diff / dist;
            // HDR exponential decay
            falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));
            if (lightType == 1) {
                // Spot cone limiter
                float theta = dot(-L, spotDirView);
                float cutOff = u_lights[i].spot_params.x;
                float outerCutOff = u_lights[i].spot_params.y;
                spotEffect = smoothstep(outerCutOff, cutOff, theta);
            }
        }

        // Diffuse adn specular calculation
        float NdotL = (lightType == 3) ? mix(1.0, max(dot(finalNormal, L), 0.0), 0.5) : max(dot(finalNormal, L), 0.0);
        float specularPower = (lightType == 3) ? 0.0 : calculateSpecular(finalNormal, L, V, isHorizontal);
        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;
        // Final accumulation
        dynamicLights += (diffuse + specular) * intensity * falloff * spotEffect;
    }

    if (u_debugLights == 1) {
        vec3 rayDir = normalize(ViewPos);
        for (int i = 0; i < u_numLights; ++i) {
            vec3 lightPosView = (u_view * vec4(u_lights[i].pos_type.xyz, 1.0)).xyz;
            float t = dot(lightPosView, rayDir);
            if (t > 0.0 && t < length(ViewPos)) {
                vec3 closestPoint = t * rayDir;
                float d = length(lightPosView - closestPoint);
                float virtualRadius = 2.0;
                if (d < virtualRadius) {
                    int lType = int(u_lights[i].pos_type.w);
                    if (lType == 1) {
                        dynamicLights = vec3(0.0, 1.0, 0.0); // Green for Spotlight
                    } else if (lType == 3 || lType == 0) {
                        dynamicLights = vec3(0.0, 0.0, 1.0); // Blue for Ambient/Point
                    }
                }
            }
        }
    }

    vec3 finalLight = litRoom + roomBeam + dynamicLights;
    FragColor = vec4(finalLight, 1.0); // Forced to 1.0 for framebuffer safety
    BrightColor = vec4(dot(finalLight, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalLight : vec3(0.0), 1.0);
}