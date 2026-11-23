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
in vec2 DetailParams;
in vec3 ViewNormal;
in vec4 ViewTangent;
in vec3 LightmapUV;
in vec4 EnvParams;

uniform sampler2DArray u_texture[4];
uniform sampler2DArray u_normalMap[4];
uniform sampler2DArray u_emissiveMap[4];
uniform samplerCube u_envMap;
uniform sampler2DArray u_lightmaps;
uniform sampler2DShadow u_shadowAtlas;
uniform usampler2D u_clusterData;

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

// Dynamic Shadow Lights (up to 8 spotlights, pre-transformed to View Space on CPU)
struct ShadowLight {
    vec4 pos_intensity;    // xyz: view pos, w: intensity
    vec4 dir_falloff;      // xyz: normalized view dir, w: falloff
    vec4 color_type;       // xyz: rgb color, w: kind
    vec4 spot_params;      // x: outer cutoff, y: inner cutoff, z: effectiveRadiusSq, w: invScale
    mat4 lightSpace;       // projection * view matrix
};

uniform ShadowLight u_shadowLights[8];
uniform int u_numShadowLights;

// UBO Dynamic point/area lights (LightsBlock binding 0, pre-transformed to View Space on CPU)
struct Light {
    vec4 pos_type;        // xyz: View position,  w: Type (0.0=Point, 1.0=Spot, 2.0=Directional, 3.0=Ambient)
    vec4 color_intensity; // xyz: RGB color,       w: Intensity
    vec4 dir_falloff;     // xyz: Normalized view direction, w: Falloff Factor
    vec4 spot_params;     // x: inner cutoff (cos), y: outer cutoff (cos), z: effectiveRadiusSq, w: invScale
};

layout(std140) uniform LightsBlock {
    Light u_lights[256];
};
uniform int u_numLights;
uniform float u_clusterNear;
uniform float u_clusterLogScale;
uniform int u_numGlobalLights;

const float PI = 3.14159265359;
const vec2 kShadowAtlasTexel = vec2(1.0 / 2048.0);

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
    float nLenSq = dot(ViewNormal, ViewNormal);
    vec3 N = (nLenSq > 1e-8) ? (ViewNormal * inversesqrt(nLenSq)) : vec3(0.0, 0.0, 1.0);
    if (!gl_FrontFacing) {
        N = -N;
    }

    vec3 mapColor = getNormal(TexCoords);
    if (dot(mapColor, mapColor) < 0.01) {
        return N;
    }
    vec3 mapNormal = normalize((mapColor * 2.0) - 1.0);

    vec3 T = ViewTangent.xyz - N * dot(N, ViewTangent.xyz);
    float tLenSq = dot(T, T);
    if (tLenSq > 1e-8) {
        T *= inversesqrt(tLenSq);
        vec3 B = cross(N, T) * ViewTangent.w;
        return normalize(mat3(T, B, N) * mapNormal);
    }
    return N;
}

float calculateSpecular(vec3 normal, vec3 lightDir, vec3 viewDir) {
    float specBoost = SpecBoost;
    float shininess = Shininess;
    if (specBoost <= 0.001 || shininess <= 0.01) {
        return 0.0;
    }
    vec3 H = normalize(lightDir + viewDir);
    float NdotH = max(dot(normal, H), 0.0);

    float energyConservation = (shininess + 2.0) / (8.0 * PI);
    return clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0) * energyConservation;
}

vec3 evalDynamicLight(int i, vec3 albedo, vec3 finalNormal, vec3 V) {
    float intensity = max(u_lights[i].color_intensity.w, 0.0);
    if (intensity <= 0.001) return vec3(0.0);

    int lightType = int(u_lights[i].pos_type.w);
    vec3 lightPosView = u_lights[i].pos_type.xyz;
    vec3 spotDirView = u_lights[i].dir_falloff.xyz;

    vec3 L;
    float falloff = 1.0;
    float spotEffect = 1.0;

    if (lightType == 2) {
        L = -spotDirView;
    } else {
        vec3 diff = lightPosView - ViewPos;
        float distSq = dot(diff, diff);
        if (distSq > u_lights[i].spot_params.z) return vec3(0.0);
        float dist = sqrt(distSq);
        if (dist <= 0.0001) return vec3(0.0);

        L = diff / dist;
        falloff = exp(-dist * u_lights[i].spot_params.w);
        if (lightType == 1) {
            float theta = dot(-L, spotDirView);
            float cutOff = u_lights[i].spot_params.x;
            float outerCutOff = u_lights[i].spot_params.y;
            spotEffect = smoothstep(outerCutOff, cutOff, theta);
            if (spotEffect <= 0.001) return vec3(0.0);
        }
    }

    float rawNdotL = dot(finalNormal, L);
    if (lightType != 3 && rawNdotL <= 0.0) return vec3(0.0);
    float NdotL = (lightType == 3) ? mix(1.0, max(rawNdotL, 0.0), 0.5) : rawNdotL;

    vec3 lightColor = u_lights[i].color_intensity.xyz;
    float specularPower = (lightType == 3) ? 0.0 : calculateSpecular(finalNormal, L, V);
    vec3 diffuse = albedo * lightColor * NdotL;
    vec3 specular = vec3(specularPower) * lightColor;
    return (diffuse + specular) * intensity * falloff * spotEffect;
}



// Procedural micro-surface detail noise generator (eliminates up-close pixelation)
float hash21(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}

float detailNoise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = hash21(i);
    float b = hash21(i + vec2(1.0, 0.0));
    float c = hash21(i + vec2(0.0, 1.0));
    float d = hash21(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float getDetailMask(vec2 uv) {
    float n1 = detailNoise(uv);
    float n2 = detailNoise(uv * 2.03 + vec2(1.7, 9.2));
    return (n1 * 0.65 + n2 * 0.35);
}

void main() {
    vec4 texColor = getDiffuse(TexCoords);
#ifdef ALPHA_CUTOUT
    if (texColor.a < 0.5) {
        discard;
    }
#endif

    vec3 albedo = pow(texColor.rgb, vec3(2.2));

    // High-frequency detail micro-texture to eliminate pixelation up close
    if (DetailParams.y > 0.001) {
        float distFade = clamp(1.0 - FragDepth / 600.0, 0.0, 1.0);
        if (distFade > 0.001) {
            float detail = getDetailMask(TexCoords.xy * DetailParams.x);
            float modFactor = mix(1.0, detail * 2.0, DetailParams.y * distFade);
            albedo *= modFactor;
        }
    }
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
    vec3 L_room_dir = normalize(u_view[1].xyz);
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
    vec3 baseLight = vec3(u_ambient_light * (0.2 + 0.8 * NdotL_room * roomShadow));
    if (LightmapUV.z >= 0.0) {
        vec3 lmSample = texture(u_lightmaps, vec3(LightmapUV.xy, floor(LightmapUV.z + 0.5))).rgb;
        vec3 lmLinear = pow(max(lmSample, vec3(0.0)), vec3(2.2));
        baseLight = mix(baseLight, max(lmLinear * 1.6, baseLight * 0.35), 0.85);
    }
    vec3 litRoom = albedo * baseLight;

    // FLASHLIGHT
    vec3 litFlash = vec3(0.0);
    if (u_enableFlashlight == 1 && u_flashIntensityFactor > 0.001) {
        vec3 flashPosView = u_flashPosView;
        vec3 flashSpotDir = normalize((u_flashDir * 512.0) - flashPosView);
        vec3 toFlash = flashPosView - ViewPos;
        vec3 L_flash = normalize(toFlash);
        float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, dot(-L_flash, flashSpotDir));

        if (flashCone > 0.001) {
            float diffFlash = max(dot(finalNormal, L_flash), 0.0);
            if (diffFlash > 0.0) {
                float distToLight = length(toFlash);
                float distanceFade = smoothstep(u_flashFalloff, u_flashFalloff * 0.8, distToLight);
                float flashIntensity = flashCone * (u_flashFalloff * u_flashIntensityFactor) * distanceFade;

                float shadowFlash = 1.0;
                if (u_enableShadows == 1 && FragPosLightFlash.w > 0.0) {
                    vec3 projFlash = FragPosLightFlash.xyz / FragPosLightFlash.w * 0.5 + 0.5;
                    if (projFlash.z >= 0.0 && projFlash.z <= 1.0 && projFlash.x >= 0.0 && projFlash.x <= 1.0 && projFlash.y >= 0.0 && projFlash.y <= 1.0) {
                        float nLenSq = dot(ViewNormal, ViewNormal);
                        vec3 geoN = (nLenSq > 1e-8) ? (ViewNormal * inversesqrt(nLenSq)) : finalNormal;
                        if (!gl_FrontFacing) {
                            geoN = -geoN;
                        }
                        float cosTheta = clamp(dot(geoN, L_flash), 0.0, 1.0);
                        float bias = max(0.00012 * (1.0 - cosTheta), 0.00002);
                        vec2 atlasUV = vec2(0.5 + clamp(projFlash.x, 0.0015, 0.9985) * 0.5, 0.5 + clamp(projFlash.y, 0.0015, 0.9985) * 0.5);
                        float s = 0.0;
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5, -0.5) * kShadowAtlasTexel, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5, -0.5) * kShadowAtlasTexel, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5,  0.5) * kShadowAtlasTexel, projFlash.z - bias));
                        s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5,  0.5) * kShadowAtlasTexel, projFlash.z - bias));
                        float shadowSample = s * 0.25;
                        float edgeFade = smoothstep(0.0, 0.03, projFlash.x) * smoothstep(1.0, 0.97, projFlash.x) *
                                         smoothstep(0.0, 0.03, projFlash.y) * smoothstep(1.0, 0.97, projFlash.y);
                        shadowFlash = mix(1.0, shadowSample, edgeFade);
                    }
                }

                float specFlash = calculateSpecular(finalNormal, L_flash, V);
                litFlash = (albedo * diffFlash + vec3(specFlash)) * (flashIntensity * 2.5) * shadowFlash * vec3(1.0, 0.98, 0.9);
            }
        }
    }

    // DYNAMIC SHADOW LIGHTS (Spotlights with Shadow Atlas)
    vec3 shadowLightsTotal = vec3(0.0);
    for (int i = 0; i < u_numShadowLights; ++i) {
        float intensity = max(u_shadowLights[i].pos_intensity.w, 0.0);
        if (intensity <= 0.001) continue;

        int lightType = int(u_shadowLights[i].color_type.w);
        vec3 lightPosView = u_shadowLights[i].pos_intensity.xyz;
        vec3 spotDirView = u_shadowLights[i].dir_falloff.xyz;

        vec3 L;
        float falloff = 1.0;
        float spotEffect = 1.0;

        if (lightType == 2) {
            L = -spotDirView;
        } else {
            vec3 diff = lightPosView - ViewPos;
            float distSq = dot(diff, diff);
            if (distSq > u_shadowLights[i].spot_params.z) continue;
            float dist = sqrt(distSq);
            if (dist <= 0.0001) continue;

            L = diff / dist;
            falloff = exp(-dist * u_shadowLights[i].spot_params.w);
            if (lightType == 1) {
                float theta = dot(-L, spotDirView);
                float outerCutOff = u_shadowLights[i].spot_params.x;
                float cutOff = u_shadowLights[i].spot_params.y;
                spotEffect = smoothstep(outerCutOff, cutOff, theta);
                if (spotEffect <= 0.001) continue;
            }
        }

        float rawNdotL = dot(finalNormal, L);
        if (lightType != 3 && rawNdotL <= 0.0) continue;
        float NdotL = (lightType == 3) ? mix(1.0, max(rawNdotL, 0.0), 0.5) : rawNdotL;

        float shadow = 1.0;
        if (u_enableShadows == 1 && lightType == 1) {
            vec4 lightClip = u_shadowLights[i].lightSpace * vec4(WorldPos, 1.0);
            if (lightClip.w > 0.0) {
                vec3 proj = lightClip.xyz / lightClip.w * 0.5 + 0.5;
                if (proj.z <= 1.0 && proj.x >= 0.0 && proj.x <= 1.0 && proj.y >= 0.0 && proj.y <= 1.0) {
                    float col = float(i % 4);
                    float row = float(i / 4);
                    vec2 atlasUV = vec2((clamp(proj.x, 0.001, 0.999) + col) * 0.25, (clamp(proj.y, 0.001, 0.999) + row) * 0.25);
                    float cosTheta = clamp(NdotL, 0.0, 1.0);
                    float bias = max(0.0025 * (1.0 - cosTheta), 0.0005);

                    float s = 0.0;
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5, -0.5) * kShadowAtlasTexel, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5, -0.5) * kShadowAtlasTexel, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2(-0.5,  0.5) * kShadowAtlasTexel, proj.z - bias));
                    s += texture(u_shadowAtlas, vec3(atlasUV + vec2( 0.5,  0.5) * kShadowAtlasTexel, proj.z - bias));
                    float shadowOcclusion = s * 0.25;
                    float edgeFade = smoothstep(0.0, 0.05, proj.x) * smoothstep(1.0, 0.95, proj.x) *
                                     smoothstep(0.0, 0.05, proj.y) * smoothstep(1.0, 0.95, proj.y);
                    shadow = mix(1.0, shadowOcclusion, edgeFade);
                }
            }
        }

        vec3 lightColor = u_shadowLights[i].color_type.xyz;
        float specularPower = (lightType == 3) ? 0.0 : calculateSpecular(finalNormal, L, V);
        vec3 diffuse = albedo * lightColor * NdotL;
        vec3 specular = vec3(specularPower) * lightColor;
        shadowLightsTotal += (diffuse + specular) * intensity * falloff * spotEffect * shadow;
    }

    // DYNAMIC POINT/AREA/DIR LIGHTS (Clustered Forward + LightsBlock)
    vec3 dynamicLightsTotal = vec3(0.0);
    if (u_numLights > 0) {
        if (u_clusterLogScale > 0.0) {
            int cx = clamp(int(floor(screenUV.x * 16.0)), 0, 15);
            int cy = clamp(int(floor(screenUV.y * 9.0)), 0, 8);
            float viewDepth = max(-ViewPos.z, max(u_clusterNear, 0.1));
            int cz = clamp(int(floor(log(viewDepth / max(u_clusterNear, 0.1)) * u_clusterLogScale)), 0, 15);
            int cid = (cz * 9 + cy) * 16 + cx;
            uvec4 header = texelFetch(u_clusterData, ivec2(cid & 127, cid >> 7), 0);

            for (int gi = 0; gi < u_numGlobalLights; ++gi) {
                uint texelLinear = 2304u + (uint(gi) >> 2u);
                uvec4 packedIdx = texelFetch(u_clusterData, ivec2(int(texelLinear & 127u), int(texelLinear >> 7u)), 0);
                uint ch = uint(gi) & 3u;
                int lightIdx = int(ch == 0u ? packedIdx.r : (ch == 1u ? packedIdx.g : (ch == 2u ? packedIdx.b : packedIdx.a)));
                dynamicLightsTotal += evalDynamicLight(lightIdx, albedo, finalNormal, V);
            }

            uint offset = header.r | (header.b << 16u);
            uint count = header.g;
            for (uint li = 0u; li < count; ++li) {
                uint slot = offset + li;
                uint texelLinear = 2304u + (slot >> 2u);
                uvec4 packedIdx = texelFetch(u_clusterData, ivec2(int(texelLinear & 127u), int(texelLinear >> 7u)), 0);
                uint ch = slot & 3u;
                int lightIdx = int(ch == 0u ? packedIdx.r : (ch == 1u ? packedIdx.g : (ch == 2u ? packedIdx.b : packedIdx.a)));
                dynamicLightsTotal += evalDynamicLight(lightIdx, albedo, finalNormal, V);
            }
        } else {
            for (int i = 0; i < u_numLights; ++i) {
                dynamicLightsTotal += evalDynamicLight(i, albedo, finalNormal, V);
            }
        }
    }

    // ENVIRONMENT CUBEMAP REFLECTION (Fresnel Schlick)
    vec3 envReflection = vec3(0.0);
    if (EnvParams.x > 0.001) {
        vec3 R_view = reflect(-V, finalNormal);
        vec3 R_world = normalize(mat3(u_invView) * R_view);
        vec3 envSample = texture(u_envMap, R_world).rgb;
        float fresnel = mix(0.04, 1.0, pow(1.0 - max(dot(finalNormal, V), 0.0), 4.0));
        float specScale = clamp(0.25 + SpecBoost * 0.5, 0.25, 1.5);
        envReflection = envSample * EnvParams.yzw * EnvParams.x * fresnel * specScale;
    }

    // EMISSIVE
    vec3 emissive = getEmissive(TexCoords) * u_emissiveIntensity * edgeFade;

    // TOTAL ACCUMULATION
    vec3 finalColor = litRoom + litFlash + shadowLightsTotal + dynamicLightsTotal + envReflection + emissive;
    float outAlpha = texColor.a;
    if (EnvParams.x > 0.001) {
        outAlpha = max(outAlpha, clamp(dot(envReflection, vec3(0.3, 0.6, 0.1)) * 0.75, 0.0, 0.85));
    }

    FragColor = vec4(finalColor, outAlpha);
    BrightColor = vec4(dot(finalColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalColor : vec3(0.0), 1.0);
}