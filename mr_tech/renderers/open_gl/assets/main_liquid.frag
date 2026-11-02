#version 330 core

in vec3 TexCoords;
in vec3 ViewPos;
in vec3 WorldPos;
in float Material;

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_texture[4];

uniform sampler2D u_refractionTex;
uniform sampler2D u_depthTex;

uniform vec2 u_resolution;
uniform float u_time;

uniform float u_near;
uniform float u_far;

uniform vec3 u_lightDir;
//uniform int u_liquidType;

const int LIQUID_WATER = 0; // Acqua limpida
const int LIQUID_SEA   = 1; // Mare con onde ampie e schiuma su creste e riva
const int LIQUID_LAVA  = 2; // Lava incandescente opaca con vene di magma
const int LIQUID_SLIME = 3; // Slime / Nukage acido radioattivo
const int LIQUID_BLOOD = 4; // Sangue rosso denso
const int LIQUID_FOG   = 5; // Nebbia volumetrica atmosferica

// Water / Liquid diffuse texture sampler
vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z + 0.5) / 1000;
    float l = floor(mod(tc.z + 0.5, 1000.0));
    if (b <= 0) {
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

// 2D Hash
float hash21(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}

// 2D Value noise
float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = hash21(i);
    float b = hash21(i + vec2(1.0, 0.0));
    float c = hash21(i + vec2(0.0, 1.0));
    float d = hash21(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

// Fractal Brownian Motion
float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    value += noise(p) * amplitude;
    p *= 2.03;
    amplitude *= 0.5;
    value += noise(p) * amplitude;
    p *= 2.01;
    amplitude *= 0.5;
    value += noise(p) * amplitude;
    return value;
}

vec3 refraction(float depthDiff, vec2 distortion, vec2 distortedUV) {
    float blurRadius = clamp(depthDiff * 0.00007, 0.0007, 0.0035);
    vec2 dispDir = length(distortion) > 1e-5 ? normalize(distortion) : vec2(0.707, 0.707);
    vec2 dispOffset = dispDir * (blurRadius * 0.85);
    vec3 cCenter = texture(u_refractionTex, distortedUV).rgb;
    vec3 c1 = texture(u_refractionTex, distortedUV + vec2(blurRadius, blurRadius * 0.55)).rgb;
    vec3 c2 = texture(u_refractionTex, distortedUV - vec2(blurRadius, blurRadius * 0.55)).rgb;
    vec3 c3 = texture(u_refractionTex, distortedUV + vec2(-blurRadius * 0.55, blurRadius)).rgb;
    vec3 c4 = texture(u_refractionTex, distortedUV + vec2(blurRadius * 0.55, -blurRadius)).rgb;
    vec3 diffusedRefraction = cCenter * 0.36 + (c1 + c2 + c3 + c4) * 0.16;
    float rCh = texture(u_refractionTex, distortedUV + dispOffset).r;
    float bCh = texture(u_refractionTex, distortedUV - dispOffset).b;
    return vec3(rCh * 0.45 + diffusedRefraction.r * 0.55, diffusedRefraction.g, bCh * 0.45 + diffusedRefraction.b * 0.55);
}

void renderFog(bool isImpenetrable) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec3 fogBase = (length(texColor) > 0.04) ? texColor : vec3(0.72, 0.76, 0.82);
    if (isImpenetrable) {
        FragColor = vec4(fogBase, 1.0);
        BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float sampleDepthRaw = texture(u_depthTex, screenUV).r;
    float sampleZN = sampleDepthRaw * 2.0 - 1.0;
    float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
    bool hasSubmerged = (sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.5);
    float depthDiff = hasSubmerged ? max(sampleDepth - waterDepth, 0.0) : 50.0;
    float fogFactor = clamp(1.0 - exp(-depthDiff * 0.015), 0.0, 1.0);
    vec3 bg = hasSubmerged ? texture(u_refractionTex, screenUV).rgb : fogBase;
    vec3 fogColor = mix(fogBase, vec3(0.72, 0.76, 0.82), 0.35);
    vec3 finalFog = mix(bg, fogColor, fogFactor * 0.92);
    FragColor = vec4(finalFog, 1.0);
    BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
}

void renderLava(bool isImpenetrable) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    float t = u_time;
    vec2 wPos = WorldPos.xz;
    float lavaT = t * 0.5;
    float n1 = fbm(wPos * 0.015 + vec2(lavaT * 0.04, -lavaT * 0.03));
    float n2 = fbm(wPos * 0.035 + vec2(-lavaT * 0.05, lavaT * 0.04));
    float lavaNoise = mix(n1, n2, 0.5);
    vec3 lavaHot   = vec3(1.35, 0.38, 0.02);
    vec3 lavaCore  = vec3(1.90, 1.25, 0.30);
    vec3 lavaCrust = (length(texColor) > 0.04) ? texColor : vec3(0.25, 0.05, 0.01);
    float tear = clamp(lavaNoise * 1.5 - 0.25, 0.0, 1.0);
    vec3 finalLava = mix(lavaCrust, lavaHot, smoothstep(0.20, 0.60, tear));
    finalLava = mix(finalLava, lavaCore, smoothstep(0.68, 0.95, tear));

    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float sampleDepthRaw = texture(u_depthTex, screenUV).r;
    float sampleZN = sampleDepthRaw * 2.0 - 1.0;
    float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
    if (!isImpenetrable && sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.1) {
        float depthDiff = max(sampleDepth - waterDepth, 0.0);
        const float LAVA_RIM = 5.0;
        if (depthDiff < LAVA_RIM) {
            float rimGrad = 1.0 - (depthDiff / LAVA_RIM);
            float rimNoise = fbm(wPos * 0.06 + vec2(lavaT * 0.15, -lavaT * 0.12));
            float sizzle = smoothstep(0.35, 0.75, rimGrad + rimNoise * 0.4);
            finalLava = mix(finalLava, vec3(1.95, 0.95, 0.18), sizzle * 0.85);
        }
    }
    float glow = dot(finalLava, vec3(0.35, 0.55, 0.10));
    vec3 bloomLava = glow > 0.85 ? (finalLava - 0.65) * 1.3 : vec3(0.0);
    FragColor = vec4(finalLava, 1.0);
    BrightColor = vec4(bloomLava, 1.0);
}

void renderSlime(bool isImpenetrable) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float t = u_time;
    vec2 wPos = WorldPos.xz;
    vec2 p = wPos * 0.025;
    vec2 rw1 = vec2(sin(p.x * 1.7 + p.y * 0.8 + t * 1.2), cos(p.y * 1.5 - p.x * 0.9 + t * 1.0));
    vec2 rw2 = vec2(sin(p.y * 2.3 + p.x * 1.3 - t * 0.85), cos(p.x * 2.1 - p.y * 1.4 - t * 1.15));
    vec2 rw3 = vec2(sin((p.x + p.y) * 3.1 + t * 0.65), cos((p.x - p.y) * 2.8 - t * 0.75));
    float ripNoise = fbm(p * 0.6 + vec2(t * 0.03, -t * 0.025));
    vec2 distortion = (rw1 * 0.50 + rw2 * 0.35 + rw3 * 0.15) * (0.0022 + ripNoise * 0.0012);

    vec3 dPdx = dFdx(ViewPos);
    vec3 dPdy = dFdy(ViewPos);
    vec3 faceNormal = cross(dPdx, dPdy);
    float lenN = length(faceNormal);
    faceNormal = lenN > 1e-6 ? faceNormal / lenN : vec3(0.0, 1.0, 0.0);
    vec3 V = normalize(-ViewPos);
    if (dot(faceNormal, V) < 0.0) {
        faceNormal = -faceNormal;
    }
    vec3 tangent = normalize(dPdx);
    vec3 bitangent = normalize(cross(faceNormal, tangent));
    vec3 waveNormal = normalize(faceNormal + tangent * (distortion.x * 25.0) + bitangent * (distortion.y * 25.0));
    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    float fresnel = 0.02 + 0.98 * pow(1.0 - NdotV, 4.0);
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);

    vec3 slimeTint = vec3(0.18, 0.85, 0.15);
    vec3 slimeSurface = (length(texColor) > 0.04) ? mix(slimeTint, texColor, 0.60) : slimeTint;
    vec3 finalSlime = slimeSurface;
    if (!isImpenetrable) {
        vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);
        float sampleDepthRaw = texture(u_depthTex, distortedUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        if (sampleDepth < waterDepth - 0.5) {
            distortedUV = screenUV;
            sampleDepthRaw = texture(u_depthTex, screenUV).r;
            sampleZN = sampleDepthRaw * 2.0 - 1.0;
            sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        }
        if (sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.5) {
            float depthDiff = max(sampleDepth - waterDepth, 0.0);
            vec3 refractionColor = refraction(depthDiff, distortion, distortedUV);
            const vec3 SLIME_EXTINCTION = vec3(0.055, 0.010, 0.050);
            vec3 transmittance = clamp(exp(-depthDiff * SLIME_EXTINCTION), 0.0, 1.0);
            vec3 transmittedBg = refractionColor * transmittance;
            vec3 shallowSlime = mix(vec3(0.18, 0.78, 0.12), texColor * 1.15, 0.50);
            vec3 deepSlime    = mix(vec3(0.03, 0.22, 0.02), texColor * 0.35, 0.60);
            float depthFactor = clamp(1.0 - exp(-depthDiff * 0.020), 0.0, 1.0);
            vec3 slimeFog = mix(shallowSlime, deepSlime, depthFactor);
            float turbidity = mix(0.35, 0.88, depthFactor);
            vec3 slimeBody = mix(transmittedBg, slimeFog, turbidity);
            float surfaceOpacity = mix(0.75, 0.95, fresnel);
            finalSlime = mix(slimeBody, slimeSurface, surfaceOpacity);
        }
    } else {
        float waveLight = max(dot(waveNormal, L), 0.0) * 0.25 + 0.75;
        finalSlime *= waveLight;
    }

    float spec = pow(NdotH, 32.0) * 0.40 * (0.30 + 0.70 * fresnel);
    finalSlime += vec3(0.8, 1.0, 0.6) * spec;
    float slimeGlow = finalSlime.g;
    vec3 bloomSlime = slimeGlow > 0.75 ? vec3(0.12, 0.55, 0.08) * (slimeGlow - 0.75) * 2.0 : vec3(0.0);
    FragColor = vec4(finalSlime, 1.0);
    BrightColor = vec4(bloomSlime, 1.0);
}

void renderBlood(bool isImpenetrable) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float t = u_time;
    vec2 wPos = WorldPos.xz;
    vec2 p = wPos * 0.025;
    vec2 rw1 = vec2(sin(p.x * 1.7 + p.y * 0.8 + t * 1.2), cos(p.y * 1.5 - p.x * 0.9 + t * 1.0));
    vec2 rw2 = vec2(sin(p.y * 2.3 + p.x * 1.3 - t * 0.85), cos(p.x * 2.1 - p.y * 1.4 - t * 1.15));
    vec2 rw3 = vec2(sin((p.x + p.y) * 3.1 + t * 0.65), cos((p.x - p.y) * 2.8 - t * 0.75));
    float ripNoise = fbm(p * 0.6 + vec2(t * 0.03, -t * 0.025));
    vec2 distortion = (rw1 * 0.50 + rw2 * 0.35 + rw3 * 0.15) * (0.0022 + ripNoise * 0.0012);

    vec3 dPdx = dFdx(ViewPos);
    vec3 dPdy = dFdy(ViewPos);
    vec3 faceNormal = cross(dPdx, dPdy);
    float lenN = length(faceNormal);
    faceNormal = lenN > 1e-6 ? faceNormal / lenN : vec3(0.0, 1.0, 0.0);
    vec3 V = normalize(-ViewPos);
    if (dot(faceNormal, V) < 0.0) {
        faceNormal = -faceNormal;
    }
    vec3 tangent = normalize(dPdx);
    vec3 bitangent = normalize(cross(faceNormal, tangent));
    vec3 waveNormal = normalize(faceNormal + tangent * (distortion.x * 25.0) + bitangent * (distortion.y * 25.0));
    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    float fresnel = 0.02 + 0.98 * pow(1.0 - NdotV, 4.0);
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);

    vec3 bloodTint = vec3(0.65, 0.05, 0.03);
    vec3 bloodSurface = (length(texColor) > 0.04) ? mix(bloodTint, texColor, 0.60) : bloodTint;
    vec3 finalBlood = bloodSurface;
    if (!isImpenetrable) {
        vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);
        float sampleDepthRaw = texture(u_depthTex, distortedUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        if (sampleDepth < waterDepth - 0.5) {
            distortedUV = screenUV;
            sampleDepthRaw = texture(u_depthTex, screenUV).r;
            sampleZN = sampleDepthRaw * 2.0 - 1.0;
            sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        }
        if (sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.5) {
            float depthDiff = max(sampleDepth - waterDepth, 0.0);
            vec3 refractionColor = refraction(depthDiff, distortion, distortedUV);
            const vec3 BLOOD_EXTINCTION = vec3(0.010, 0.055, 0.055);
            vec3 transmittance = clamp(exp(-depthDiff * BLOOD_EXTINCTION), 0.0, 1.0);
            vec3 transmittedBg = refractionColor * transmittance;
            vec3 shallowBlood = mix(vec3(0.55, 0.05, 0.03), texColor * 1.10, 0.45);
            vec3 deepBlood    = mix(vec3(0.12, 0.01, 0.01), texColor * 0.30, 0.60);
            float depthFactor = clamp(1.0 - exp(-depthDiff * 0.020), 0.0, 1.0);
            vec3 bloodFog = mix(shallowBlood, deepBlood, depthFactor);
            float turbidity = mix(0.40, 0.90, depthFactor);
            vec3 bloodBody = mix(transmittedBg, bloodFog, turbidity);
            float surfaceOpacity = mix(0.80, 0.96, fresnel);
            finalBlood = mix(bloodBody, bloodSurface, surfaceOpacity);
        }
    } else {
        float waveLight = max(dot(waveNormal, L), 0.0) * 0.25 + 0.75;
        finalBlood *= waveLight;
    }

    float spec = pow(NdotH, 48.0) * 0.50 * (0.35 + 0.65 * fresnel);
    finalBlood += vec3(1.0, 0.85, 0.85) * spec;
    FragColor = vec4(finalBlood, 1.0);
    BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
}

// --- GTA V Clear Realistic Water Simulation ---

// Multi-scale organic wave interference (calibrated for game world units, no parallel zebra lines)
vec2 getWaterWaves(vec2 pos, float t, bool isSea) {
    vec2 p = pos * (isSea ? 0.012 : 0.020);

    // Wave train 1: Rolling swell
    vec2 w1 = vec2(
        sin(p.x * 1.4 + p.y * 0.9 + t * 1.1),
        cos(p.y * 1.3 - p.x * 0.8 + t * 0.95)
    );

    // Wave train 2: Angled cross swell
    vec2 w2 = vec2(
        sin(p.y * 2.1 + p.x * 1.2 - t * 1.25),
        cos(p.x * 1.8 - p.y * 1.1 - t * 1.05)
    );

    // Wave train 3: Fine surface ripples
    vec2 w3 = vec2(
        sin((p.x + p.y) * 3.2 + t * 1.7),
        cos((p.x - p.y) * 2.9 - t * 1.5)
    );

    // Organic micro turbulence
    float turb = fbm(p * 1.6 + vec2(t * 0.05, -t * 0.04));

    vec2 combined = w1 * 0.45 + w2 * 0.35 + w3 * 0.20;
    combined += (turb - 0.5) * 0.25;
    return combined;
}

// GTA V Procedural Dancing Sunlight Caustics
float gtaCaustics(vec2 pos, float t) {
    vec2 p = pos * 0.045;
    vec2 p1 = p + vec2(t * 0.05, t * 0.03);
    vec2 p2 = p * 1.3 - vec2(t * 0.04, -t * 0.06);

    float c1 = sin(p1.x * 2.2 + cos(p1.y * 1.9 + t * 0.6)) * 0.5 + 0.5;
    float c2 = cos(p2.y * 2.4 + sin(p2.x * 2.0 - t * 0.5)) * 0.5 + 0.5;
    float c = c1 * c2;
    c = pow(c, 2.0) * 2.2;
    return clamp(c, 0.0, 2.5);
}

void renderGTAWaterInternal(bool isImpenetrable, bool isSea) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float t = u_time;
    vec2 wPos = WorldPos.xz;

    // Organic wave field
    vec2 waveDisp = getWaterWaves(wPos, t, isSea);

    // Surface normal perturbation (gentle and smooth, preventing harsh stripes)
    vec3 dPdx = dFdx(ViewPos);
    vec3 dPdy = dFdy(ViewPos);
    vec3 faceNormal = cross(dPdx, dPdy);
    float lenN = length(faceNormal);
    faceNormal = lenN > 1e-6 ? faceNormal / lenN : vec3(0.0, 1.0, 0.0);
    vec3 V = normalize(-ViewPos);
    if (dot(faceNormal, V) < 0.0) {
        faceNormal = -faceNormal;
    }
    vec3 tangent = normalize(dPdx);
    vec3 bitangent = normalize(cross(faceNormal, tangent));

    float waveScale = isSea ? 0.30 : 0.16;
    vec3 waveNormal = normalize(faceNormal + tangent * (waveDisp.x * waveScale) + bitangent * (waveDisp.y * waveScale));

    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    // Subtle Fresnel: water remains translucent from all viewing angles
    float fresnel = 0.02 + 0.48 * pow(1.0 - NdotV, 4.0);

    // Sunlight and reflection vectors
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);
    vec3 R = reflect(-V, waveNormal);

    // GTA V Sky gradient reflection
    vec3 skyHorizon    = vec3(0.55, 0.72, 0.88);
    vec3 skyZenith     = vec3(0.22, 0.46, 0.75);
    vec3 envReflection = mix(skyHorizon, skyZenith, clamp(R.y * 0.5 + 0.5, 0.0, 1.0));

    // GTA V Vibrant Tropical Water Colors (Clear cyan-blue, NEVER pitch black)
    vec3 shallowWaterColor = vec3(0.16, 0.74, 0.84);
    vec3 midWaterColor     = vec3(0.07, 0.46, 0.64);
    vec3 deepWaterColor    = vec3(0.03, 0.26, 0.46);

    vec3 finalWater;

    if (isImpenetrable) {
        // --- Impenetrable Water (Doom, Duke3D, Jedi floor liquids) ---
        vec2 refractedUV = TexCoords.xy + waveDisp * 0.018;
        vec3 bedTex = getDiffuse(vec3(refractedUV, TexCoords.z)).rgb;
        vec3 bedBase = (length(bedTex) > 0.03) ? bedTex : vec3(0.20, 0.45, 0.52);

        // Dancing caustics on the pool bed
        float caustics = gtaCaustics(wPos + waveDisp * 15.0, t);
        vec3 illuminatedBed = bedBase * (0.90 + caustics * vec3(0.32, 0.62, 0.58));

        // Translucent water volume tint
        vec3 waterBody = mix(illuminatedBed, illuminatedBed * shallowWaterColor * 1.35 + midWaterColor * 0.22, 0.35);

        // Sky reflection
        finalWater = mix(waterBody, envReflection, fresnel * 0.38);
    } else {
        // --- Penetrable / Swimmable Water (Quake swimmable volumes) ---
        vec2 distortion = waveDisp * (isSea ? 0.0035 : 0.0020);
        vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);
        float sampleDepthRaw = texture(u_depthTex, distortedUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));

        if (sampleDepth < waterDepth - 0.5) {
            distortedUV = screenUV;
            sampleDepthRaw = texture(u_depthTex, screenUV).r;
            sampleZN = sampleDepthRaw * 2.0 - 1.0;
            sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        }

        if (sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.1) {
            float depthDiff = max(sampleDepth - waterDepth, 0.0);
            vec3 refractionColor = refraction(depthDiff, distortion, distortedUV);

            // Underwater dancing caustics on submerged floors & objects
            float caustics = gtaCaustics(wPos + waveDisp * 18.0, t);
            float causticFade = clamp(1.0 - depthDiff * 0.006, 0.0, 1.0);
            refractionColor += refractionColor * caustics * vec3(0.28, 0.58, 0.54) * causticFade;

            // Physical transmittance (calibrated so underwater structures stay crisp and visible)
            vec3 transmittance = clamp(exp(-depthDiff * vec3(0.0012, 0.0004, 0.0001)), 0.18, 1.0);
            vec3 transmitted = refractionColor * transmittance;

            // Crystal-clear tropical water depth tint
            float depthFactor = clamp(1.0 - exp(-depthDiff * 0.0035), 0.0, 0.60);
            vec3 waterTint = mix(shallowWaterColor, deepWaterColor, depthFactor);
            vec3 waterBody = mix(transmitted, transmitted * waterTint * 1.35 + waterTint * 0.18, depthFactor * 0.65);

            // Shoreline / Contact soft edge foam
            float shoreFoam = clamp(1.0 - depthDiff / 6.0, 0.0, 1.0);
            shoreFoam = pow(shoreFoam, 2.0) * (0.5 + 0.5 * sin(t * 3.0 + wPos.x * 0.15));

            // Sky reflection
            finalWater = mix(waterBody, envReflection, fresnel * 0.40);

            // Apply soft shoreline foam
            finalWater = mix(finalWater, vec3(0.95, 0.98, 1.0), shoreFoam * 0.55);
        } else {
            // Surface fallback
            vec3 waterBody = mix(shallowWaterColor, midWaterColor, 0.5);
            finalWater = mix(waterBody, envReflection, fresnel * 0.40);
        }
    }

    // Specular sunlight glints (GTA V sun path)
    float specBroad = pow(NdotH, 24.0) * 0.35;
    float specCore  = pow(NdotH, 128.0) * 1.5;
    vec3 sunSpecular = vec3(1.0, 0.98, 0.92) * (specBroad + specCore);
    finalWater += sunSpecular;

    FragColor = vec4(finalWater, 1.0);
    // Write specular glints to bloom channel
    float bloomLum = dot(sunSpecular, vec3(0.2126, 0.7152, 0.0722));
    vec3 bloomColor = bloomLum > 0.8 ? (sunSpecular - 0.8) * 1.2 : vec3(0.0);
    BrightColor = vec4(bloomColor, 1.0);
}

void renderSea(bool isImpenetrable) {
    renderGTAWaterInternal(isImpenetrable, true);
}

void renderWater(bool isImpenetrable) {
    renderGTAWaterInternal(isImpenetrable, false);
}

void main() {
    bool isImpenetrable = (Material >= 9.5);
    int liquidType = 0;
    if (Material >= -0.5) {
        if (isImpenetrable) {
            liquidType = int(Material - 10.0 + 0.1);
        } else {
            liquidType = int(Material + 0.1);
        }
    }

    if (liquidType == LIQUID_WATER) {
        renderWater(isImpenetrable);
    } else if (liquidType == LIQUID_SEA) {
        renderSea(isImpenetrable);
    } else if (liquidType == LIQUID_LAVA) {
        renderLava(isImpenetrable);
    } else if (liquidType == LIQUID_SLIME) {
        renderSlime(isImpenetrable);
    } else if (liquidType == LIQUID_BLOOD) {
        renderBlood(isImpenetrable);
    } else if (liquidType == LIQUID_FOG) {
        renderFog(isImpenetrable);
    } else {
        renderWater(isImpenetrable);
    }
}
