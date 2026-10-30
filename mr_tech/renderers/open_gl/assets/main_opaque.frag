#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 WorldPos;
in vec4 FragPosLightRoom;
in vec4 FragPosLightFlash;
in float IsFullbright;
in float SpecBoost;
in float Shininess;

uniform sampler2DArray u_texture[4];
uniform sampler2DArray u_normalMap[4];
uniform sampler2DArray u_emissiveMap[4];
uniform sampler2DShadow u_shadowAtlas;

uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

uniform vec2 u_screenResolution;
uniform float u_emissiveIntensity;
uniform float u_ambient_light;
uniform int u_enableShadows;

uniform int u_debugLights;

// Flashlight
uniform int u_enableFlashlight;
uniform vec3 u_flashPosView;
uniform vec3 u_flashDir;
uniform float u_flashIntensityFactor;
uniform float u_flashFalloff;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;

// Dynamic Shadow Lights (up to 8 spotlights)
struct ShadowLight {
    vec4 pos_intensity;    // xyz: world pos, w: intensity
    vec4 dir_falloff;      // xyz: world dir, w: falloff
    vec4 color_type;       // xyz: rgb color, w: kind
    vec4 spot_params;      // x: outer cutoff, y: inner cutoff, z,w: padding
    mat4 lightSpace;       // projection * view matrix
};

uniform ShadowLight u_shadowLights[8];
uniform int u_numShadowLights;

// UBO Dynamic point/area lights (LightsBlock binding 0)
struct Light {
    vec4 pos_type;        // xyz: World position,  w: Type (0.0=Point, 1.0=Spot, 2.0=Directional, 3.0=Ambient)
    vec4 color_intensity; // xyz: RGB color,       w: Intensity
    vec4 dir_falloff;     // xyz: World direction, w: Falloff Factor
    vec4 spot_params;     // x: inner cutoff (cos), y: outer cutoff (cos), z,w: padding
};

layout(std140) uniform LightsBlock {
    Light u_lights[256];
};
uniform int u_numLights;

const float PI = 3.14159265359;

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_texture[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_texture[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_texture[2], vec3(tc.xy, l));
    return texture(u_texture[3], vec3(tc.xy, l));
}

vec3 getNormal(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_normalMap[0], vec3(tc.xy, l)).rgb;
    if (b == 1) return texture(u_normalMap[1], vec3(tc.xy, l)).rgb;
    if (b == 2) return texture(u_normalMap[2], vec3(tc.xy, l)).rgb;
    return texture(u_normalMap[3], vec3(tc.xy, l)).rgb;
}

vec3 getEmissive(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_emissiveMap[0], vec3(tc.xy, l)).rgb;
    if (b == 1) return texture(u_emissiveMap[1], vec3(tc.xy, l)).rgb;
    if (b == 2) return texture(u_emissiveMap[2], vec3(tc.xy, l)).rgb;
    return texture(u_emissiveMap[3], vec3(tc.xy, l)).rgb;
}

vec3 calculateNormal() {
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec3 geoNormal = normalize(cross(dp1, dp2));
    if (geoNormal.z < 0.0) {
        geoNormal = -geoNormal;
    }
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
        T /= lenT;
        T = normalize(T - geoNormal * dot(geoNormal, T));
        B = normalize(cross(geoNormal, T));
        mat3 TBN = mat3(T, B, geoNormal);
        return normalize(TBN * mapNormal);
    }
    return geoNormal;
}

float calculateSpecular(vec3 normal, vec3 lightDir, vec3 viewDir) {
    vec3 H = normalize(lightDir + viewDir);
    float NdotH = max(dot(normal, H), 0.0);
    float shininess = Shininess;
    float specBoost = SpecBoost;

    if (shininess <= 0.01) {
        return 0.0;
    }

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

void main() {
    vec4 texColor = getDiffuse(TexCoords);
    if (texColor.a < 0.5) {
        discard;
    }

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    // Fullbright geometry bypasses all lighting calculations
    if (IsFullbright > 0.5) {
        vec3 fullbrightColor = albedo + getEmissive(TexCoords) * u_emissiveIntensity * edgeFade;
        FragColor = vec4(fullbrightColor, texColor.a);
        BrightColor = vec4(dot(fullbrightColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? fullbrightColor : vec3(0.0), 1.0);
        return;
    }

    vec3 finalNormal = calculateNormal();
    vec3 V = normalize(-ViewPos);

    //  AMBIENT & ROOM LIGHT
    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    float roomShadow = 1.0;
    if (u_enableShadows == 1 && FragPosLightRoom.w > 0.0) {
        vec3 projRoom = FragPosLightRoom.xyz / FragPosLightRoom.w * 0.5 + 0.5;
        if (projRoom.z <= 1.0 && projRoom.x >= 0.0 && projRoom.x <= 1.0 && projRoom.y >= 0.0 && projRoom.y <= 1.0) {
            float roomBias = max(0.005 * (1.0 - NdotL_room), 0.001);
            vec2 atlasUV = vec2(clamp(projRoom.x, 0.001, 0.999) * 0.5, 0.5 + clamp(projRoom.y, 0.001, 0.999) * 0.5);
            roomShadow = texture(u_shadowAtlas, vec3(atlasUV, projRoom.z - roomBias));
        }
    }
    vec3 litRoom = albedo * (u_ambient_light * (0.2 + 0.8 * NdotL_room * roomShadow));

    // FLASHLIGHT
    vec3 litFlash = vec3(0.0);
    if (u_enableFlashlight == 1 && u_flashIntensityFactor > 0.001) {
        vec3 flashPosView = u_flashPosView;
        vec3 flashSpotDir = normalize((u_flashDir * 512.0) - flashPosView);
        vec3 L_flash = normalize(flashPosView - ViewPos);
        float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-L_flash, flashSpotDir));

        if (flashCone > 0.001) {
            float distToLight = length(flashPosView - ViewPos);
            float distanceFade = smoothstep(u_flashFalloff, u_flashFalloff * 0.8, distToLight);
            float flashIntensity = flashCone * (u_flashFalloff * u_flashIntensityFactor) * distanceFade;

            float shadowFlash = 1.0;
            if (u_enableShadows == 1) {
                if (FragPosLightFlash.w > 0.0) {
                    vec3 projFlash = FragPosLightFlash.xyz / FragPosLightFlash.w * 0.5 + 0.5;
                    if (projFlash.z <= 1.0 && projFlash.x >= 0.0 && projFlash.x <= 1.0 && projFlash.y >= 0.0 && projFlash.y <= 1.0) {
                        float cosTheta = clamp(dot(finalNormal, L_flash), 0.0, 1.0);
                        float bias = max(0.0005 * (1.0 - cosTheta), 0.00005);
                        vec2 atlasUV = vec2(0.5 + clamp(projFlash.x, 0.001, 0.999) * 0.5, 0.5 + clamp(projFlash.y, 0.001, 0.999) * 0.5);
                        vec2 texelSize = 1.0 / vec2(textureSize(u_shadowAtlas, 0));
                        float s = 0.0;
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5, -0.5) * texelSize, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5, -0.5) * texelSize, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5,  0.5) * texelSize, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5,  0.5) * texelSize, projFlash.z - bias));
                        shadowFlash = s * 0.25;
                    } else {
                        shadowFlash = 0.0; // Outside shadow map frustum -> no light!
                    }
                } else {
                    shadowFlash = 0.0;
                }
            }

            float diffFlash = max(dot(finalNormal, L_flash), 0.0);
            float specFlash = calculateSpecular(finalNormal, L_flash, V);
            litFlash = (albedo * diffFlash + vec3(specFlash)) * (flashIntensity * 2.5) * shadowFlash * vec3(1.0, 0.98, 0.9);
        }
    }

    // DYNAMIC SHADOW LIGHTS (Spotlights with Shadow Atlas)
    vec3 shadowLightsTotal = vec3(0.0);
    for (int i = 0; i < u_numShadowLights; ++i) {
        vec3 lightPosWorld = u_shadowLights[i].pos_intensity.xyz;
        float intensity = u_shadowLights[i].pos_intensity.w;
        if (intensity <= 0.001) continue;

        vec3 lightDirWorld = u_shadowLights[i].dir_falloff.xyz;
        float falloffFactor = max(u_shadowLights[i].dir_falloff.w, 1.0);
        vec3 lightColor = u_shadowLights[i].color_type.xyz;
        float outerCutOff = u_shadowLights[i].spot_params.x;
        float cutOff = u_shadowLights[i].spot_params.y;

        vec3 lightPosView = (u_view * vec4(lightPosWorld, 1.0)).xyz;
        vec3 spotDirView = normalize(mat3(u_view) * lightDirWorld);

        vec3 diff = lightPosView - ViewPos;
        float dist = length(diff);
        float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);
        if (dist > effectiveRadius) continue;

        vec3 L = diff / dist;
        float theta = dot(-L, spotDirView);
        float spotEffect = smoothstep(outerCutOff, cutOff, theta);
        if (spotEffect <= 0.001) continue;

        float expFalloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));

        float shadow = 1.0;
        if (u_enableShadows == 1) {
            vec4 lightClip = u_shadowLights[i].lightSpace * vec4(WorldPos, 1.0);
            if (lightClip.w > 0.0) {
                vec3 proj = lightClip.xyz / lightClip.w * 0.5 + 0.5;
                if (proj.z <= 1.0 && proj.x >= 0.0 && proj.x <= 1.0 && proj.y >= 0.0 && proj.y <= 1.0) {
                    float col = float(i % 4);
                    float row = float(i / 4);
                    vec2 atlasUV = vec2((clamp(proj.x, 0.001, 0.999) + col) * 0.25, (clamp(proj.y, 0.001, 0.999) + row) * 0.25);
                    float cosTheta = clamp(dot(finalNormal, L), 0.0, 1.0);
                    float bias = max(0.001 * (1.0 - cosTheta), 0.0001);

                    vec2 texelSize = 1.0 / vec2(textureSize(u_shadowAtlas, 0));
                    float s = 0.0;
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5, -0.5) * texelSize, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5, -0.5) * texelSize, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5,  0.5) * texelSize, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5,  0.5) * texelSize, proj.z - bias));
                    shadow = s * 0.25;
                } else {
                    shadow = 0.0;
                }
            } else {
                shadow = 0.0;
            }
        }

        float NdotL = max(dot(finalNormal, L), 0.0);
        float spec = calculateSpecular(finalNormal, L, V);
        shadowLightsTotal += (albedo * lightColor * NdotL + vec3(spec) * lightColor) * (intensity * expFalloff * spotEffect * shadow);
    }

    // DYNAMIC POINT/AREA/DIR LIGHTS (UBO LightsBlock)
    vec3 dynamicLightsTotal = vec3(0.0);
    for (int i = 0; i < u_numLights; ++i) {
        int lightType = int(u_lights[i].pos_type.w);
        float intensity = max(u_lights[i].color_intensity.w, 0.0);
        if (intensity <= 0.001) continue;

        vec3 lightColor = u_lights[i].color_intensity.xyz;
        vec3 lightPosView = (u_view * vec4(u_lights[i].pos_type.xyz, 1.0)).xyz;
        vec3 spotDirView = normalize(mat3(u_view) * u_lights[i].dir_falloff.xyz);
        float falloffFactor = max(u_lights[i].dir_falloff.w, 1.0);

        vec3 L;
        float dist;
        float falloff = 1.0;
        float spotEffect = 1.0;

        if (lightType == 2) {
            L = -spotDirView;
        } else {
            vec3 diff = lightPosView - ViewPos;
            dist = length(diff);
            float effectiveRadius = 4.605 * falloffFactor * max(intensity, 0.1);
            if (dist > effectiveRadius) continue;

            L = diff / dist;
            falloff = exp(-dist / (falloffFactor * max(intensity, 0.1)));
            if (lightType == 1) {
                float theta = dot(-L, spotDirView);
                float cutOff = u_lights[i].spot_params.x;
                float outerCutOff = u_lights[i].spot_params.y;
                spotEffect = smoothstep(outerCutOff, cutOff, theta);
            }
        }

        float NdotL = (lightType == 3) ? mix(1.0, max(dot(finalNormal, L), 0.0), 0.5) : max(dot(finalNormal, L), 0.0);
        float specularPower = (lightType == 3) ? 0.0 : calculateSpecular(finalNormal, L, V);
        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;
        dynamicLightsTotal += (diffuse + specular) * intensity * falloff * spotEffect;
    }

    // EMISSIVE
    vec3 emissive = getEmissive(TexCoords) * u_emissiveIntensity * edgeFade;

    // TOTAL ACCUMULATION
    vec3 finalColor = litRoom + litFlash + shadowLightsTotal + dynamicLightsTotal + emissive;

    FragColor = vec4(finalColor, texColor.a);
    BrightColor = vec4(dot(finalColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalColor : vec3(0.0), 1.0);
}