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

uniform mat4 u_view;
uniform vec3 u_lightDir;

const uint LIQUID_WATER = 2u; // Acqua limpida
const uint LIQUID_SEA   = 3u; // Mare con onde ampie e schiuma su creste e riva
const uint LIQUID_LAVA  = 4u; // Lava incandescente opaca con vene di magma
const uint LIQUID_SLIME = 5u; // Slime / Nukage acido radioattivo
const uint LIQUID_BLOOD = 6u; // Sangue rosso denso
const uint LIQUID_FOG   = 7u; // Nebbia volumetrica atmosferica


// TODO FROM CONFIG
// Water Colors (Lush Caribbean turquoise to deep sapphire blue)
const vec3 waterColorShallow = vec3(0.10, 0.62, 0.82);
const vec3 waterColorDeep    = vec3(0.03, 0.30, 0.52);
// Environment reflection (Lush sky & ambient sheen)
const vec3 envHorizon    = vec3(0.42, 0.68, 0.88);
const vec3 envZenith     = vec3(0.18, 0.38, 0.65);



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

void renderFog(bool isSolid) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec3 fogBase = (length(texColor) > 0.04) ? texColor : vec3(0.72, 0.76, 0.82);
    if (isSolid) {
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

void renderLava(bool isSolid) {
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
    if (!isSolid && sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.1) {
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

void renderSlime(bool isSolid) {
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

    vec3 dWdx = dFdx(WorldPos);
    vec3 dWdy = dFdy(WorldPos);
    vec3 geomNormal = cross(dWdx, dWdy);
    float lenGN = length(geomNormal);
    geomNormal = lenGN > 1e-6 ? geomNormal / lenGN : vec3(0.0, 1.0, 0.0);
    vec3 viewDir = normalize(-ViewPos);
    if (dot(mat3(u_view) * geomNormal, viewDir) < 0.0) {
        geomNormal = -geomNormal;
    }
    bool isHorizontal = abs(geomNormal.y) > 0.6;
    vec3 worldFaceNormal = isHorizontal ? vec3(0.0, 1.0, 0.0) : geomNormal;
    if (dot(mat3(u_view) * worldFaceNormal, viewDir) < 0.0) {
        worldFaceNormal = -worldFaceNormal;
    }

    uint packedMaterial = floatBitsToUint(Material);
    float vertAmp = float((packedMaterial >> 16u) & 0xFFu);
    float vertSlopeX = isHorizontal ? (0.05 * cos(WorldPos.x * 0.05 + t * 2.0) * vertAmp) : 0.0;
    float vertSlopeZ = isHorizontal ? (-0.05 * sin(WorldPos.z * 0.05 + t * 1.5) * vertAmp) : 0.0;
    vec2 totalSlope = distortion * 25.0 + vec2(vertSlopeX, vertSlopeZ);
    vec3 worldWaveNormal;
    if (isHorizontal) {
        worldWaveNormal = normalize(vec3(-totalSlope.x, worldFaceNormal.y, -totalSlope.y));
    } else {
        vec3 tAxis = abs(worldFaceNormal.y) > 0.9 ? vec3(1.0, 0.0, 0.0) : vec3(0.0, 1.0, 0.0);
        vec3 t1 = normalize(cross(tAxis, worldFaceNormal));
        vec3 t2 = cross(worldFaceNormal, t1);
        worldWaveNormal = normalize(worldFaceNormal + t1 * distortion.x * 25.0 + t2 * distortion.y * 25.0);
    }
    vec3 waveNormal = normalize(mat3(u_view) * worldWaveNormal);
    vec3 V = viewDir;
    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    float fresnel = 0.02 + 0.98 * pow(1.0 - NdotV, 4.0);
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);

    vec3 slimeTint = vec3(0.18, 0.85, 0.15);
    vec3 slimeSurface = (length(texColor) > 0.04) ? mix(slimeTint, texColor, 0.60) : slimeTint;
    vec3 finalSlime = slimeSurface;
    if (!isSolid) {
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
        vec2 refractedUV = TexCoords.xy + distortion * 2.0;
        vec3 bedTex = getDiffuse(vec3(refractedUV, TexCoords.z)).rgb;
        vec3 bedBase = (length(bedTex) > 0.03) ? bedTex : slimeTint;
        float waveLight = max(dot(waveNormal, L), 0.0) * 0.25 + 0.75;
        finalSlime = mix(bedBase * waveLight, slimeSurface, 0.45);
    }

    float spec = pow(NdotH, 32.0) * 0.40 * (0.30 + 0.70 * fresnel);
    finalSlime += vec3(0.8, 1.0, 0.6) * spec;
    float slimeGlow = finalSlime.g;
    vec3 bloomSlime = slimeGlow > 0.75 ? vec3(0.12, 0.55, 0.08) * (slimeGlow - 0.75) * 2.0 : vec3(0.0);
    FragColor = vec4(finalSlime, 1.0);
    BrightColor = vec4(bloomSlime, 1.0);
}

void renderBlood(bool isSolid) {
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

    vec3 dWdx = dFdx(WorldPos);
    vec3 dWdy = dFdy(WorldPos);
    vec3 geomNormal = cross(dWdx, dWdy);
    float lenGN = length(geomNormal);
    geomNormal = lenGN > 1e-6 ? geomNormal / lenGN : vec3(0.0, 1.0, 0.0);
    vec3 viewDir = normalize(-ViewPos);
    if (dot(mat3(u_view) * geomNormal, viewDir) < 0.0) {
        geomNormal = -geomNormal;
    }
    bool isHorizontal = abs(geomNormal.y) > 0.6;
    vec3 worldFaceNormal = isHorizontal ? vec3(0.0, 1.0, 0.0) : geomNormal;
    if (dot(mat3(u_view) * worldFaceNormal, viewDir) < 0.0) {
        worldFaceNormal = -worldFaceNormal;
    }

    uint packedMaterial = floatBitsToUint(Material);
    float vertAmp = float((packedMaterial >> 16u) & 0xFFu);
    float vertSlopeX = isHorizontal ? (0.05 * cos(WorldPos.x * 0.05 + t * 2.0) * vertAmp) : 0.0;
    float vertSlopeZ = isHorizontal ? (-0.05 * sin(WorldPos.z * 0.05 + t * 1.5) * vertAmp) : 0.0;
    vec2 totalSlope = distortion * 25.0 + vec2(vertSlopeX, vertSlopeZ);
    vec3 worldWaveNormal;
    if (isHorizontal) {
        worldWaveNormal = normalize(vec3(-totalSlope.x, worldFaceNormal.y, -totalSlope.y));
    } else {
        vec3 tAxis = abs(worldFaceNormal.y) > 0.9 ? vec3(1.0, 0.0, 0.0) : vec3(0.0, 1.0, 0.0);
        vec3 t1 = normalize(cross(tAxis, worldFaceNormal));
        vec3 t2 = cross(worldFaceNormal, t1);
        worldWaveNormal = normalize(worldFaceNormal + t1 * distortion.x * 25.0 + t2 * distortion.y * 25.0);
    }
    vec3 waveNormal = normalize(mat3(u_view) * worldWaveNormal);
    vec3 V = viewDir;
    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    float fresnel = 0.02 + 0.98 * pow(1.0 - NdotV, 4.0);
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);

    vec3 bloodTint = vec3(0.65, 0.05, 0.03);
    vec3 bloodSurface = (length(texColor) > 0.04) ? mix(bloodTint, texColor, 0.60) : bloodTint;
    vec3 finalBlood = bloodSurface;
    if (!isSolid) {
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
        vec2 refractedUV = TexCoords.xy + distortion * 2.0;
        vec3 bedTex = getDiffuse(vec3(refractedUV, TexCoords.z)).rgb;
        vec3 bedBase = (length(bedTex) > 0.03) ? bedTex : bloodTint;
        float waveLight = max(dot(waveNormal, L), 0.0) * 0.25 + 0.75;
        finalBlood = mix(bedBase * waveLight, bloodSurface, 0.45);
    }

    float spec = pow(NdotH, 48.0) * 0.50 * (0.35 + 0.65 * fresnel);
    finalBlood += vec3(1.0, 0.85, 0.85) * spec;
    FragColor = vec4(finalBlood, 1.0);
    BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
}

// Multi-scale Gerstner wave spectrum with camera distance LOD filtering
vec2 getWaterWaves(vec2 pos, float t, float viewDist, bool isSea) {
    float distFade1 = clamp(1.0 - viewDist / 600.0, 0.0, 1.0);
    float distFade2 = clamp(1.0 - viewDist / 350.0, 0.0, 1.0);
    float scale = isSea ? 1.6 : 1.0;

    vec2 totalSlope = vec2(0.0);

    // Wave 1: Primary rolling swell (direction ~ 22 deg)
    vec2 d1 = vec2(0.927, 0.375);
    float p1 = dot(d1, pos) * (0.075 / scale) - t * 1.1;
    totalSlope += d1 * (0.042 * cos(p1));

    // Wave 2: Counter rolling swell (direction ~ 112 deg)
    vec2 d2 = vec2(-0.375, 0.927);
    float p2 = dot(d2, pos) * (0.120 / scale) - t * 1.35;
    totalSlope += d2 * (0.035 * cos(p2));

    // Wave 3: Cross chop (direction ~ 213 deg)
    vec2 d3 = vec2(-0.839, -0.545);
    float p3 = dot(d3, pos) * (0.195 / scale) + t * 1.6;
    totalSlope += d3 * (0.028 * cos(p3));

    // Wave 4: Diagonal ripple (direction ~ 303 deg)
    vec2 d4 = vec2(0.545, -0.839);
    float p4 = dot(d4, pos) * (0.340 / scale) - t * 1.9;
    totalSlope += d4 * (0.022 * cos(p4));

    // Wave 5: Medium wind ripple (direction ~ 45 deg, fades with distance to prevent moiré)
    vec2 d5 = vec2(0.707, 0.707);
    float p5 = dot(d5, pos) * (0.580 / scale) + t * 2.3;
    totalSlope += d5 * (0.018 * cos(p5) * distFade1);

    // Wave 6: Medium chop (direction ~ 156 deg)
    vec2 d6 = vec2(-0.914, 0.407);
    float p6 = dot(d6, pos) * (0.920 / scale) - t * 2.8;
    totalSlope += d6 * (0.014 * cos(p6) * distFade1);

    // Wave 7: Capillary surface ripple (direction ~ 258 deg, fades at distance)
    vec2 d7 = vec2(-0.208, -0.978);
    float p7 = dot(d7, pos) * (1.650 / scale) + t * 3.4;
    totalSlope += d7 * (0.010 * cos(p7) * distFade2);

    // Wave 8: Micro capillary shimmer (direction ~ 340 deg)
    vec2 d8 = vec2(0.940, -0.342);
    float p8 = dot(d8, pos) * (2.850 / scale) - t * 4.1;
    totalSlope += d8 * (0.007 * cos(p8) * distFade2);

    return totalSlope * (isSea ? 1.4 : 1.0);
}

// procedural Dancing Sunlight Caustics
float dancingSunlightCaustics(vec2 pos, float t) {
    vec2 p = pos * 0.035;
    vec2 p1 = p + vec2(t * 0.04, t * 0.02);
    vec2 p2 = p * 1.25 - vec2(t * 0.03, -t * 0.05);
    float c1 = sin(p1.x * 2.2 + cos(p1.y * 1.9 + t * 0.5)) * 0.5 + 0.5;
    float c2 = cos(p2.y * 2.5 + sin(p2.x * 2.1 - t * 0.5)) * 0.5 + 0.5;
    float c = c1 * c2;
    return pow(c, 2.0) * 1.8;
}

void renderWater(bool isSolid, bool isSea) {
    vec3 texColor = getDiffuse(TexCoords).rgb;
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float t = u_time;
    vec2 wPos = WorldPos.xz;
    float viewDist = length(ViewPos);
    // Multi-scale wave field with distance filtering
    vec2 waveDisp = getWaterWaves(wPos, t, viewDist, isSea);
    // Compute seamless normal and tangent frame in world coordinates.
    // This guarantees that adjacent triangles forming flat water quads share the exact same
    // orientation and normal derivatives, completely eliminating visible triangle seams under specular light.
    vec3 dWdx = dFdx(WorldPos);
    vec3 dWdy = dFdy(WorldPos);
    vec3 geomNormal = cross(dWdx, dWdy);
    float lenGN = length(geomNormal);
    geomNormal = lenGN > 1e-6 ? geomNormal / lenGN : vec3(0.0, 1.0, 0.0);
    // Orientation check against view direction
    vec3 viewDir = normalize(-ViewPos);
    if (dot(mat3(u_view) * geomNormal, viewDir) < 0.0) {
        geomNormal = -geomNormal;
    }
    // For horizontal liquid surfaces (standard water/pool/sea), use fixed up normal (0,1,0)
    // combined with analytic vertex wave slope to guarantee 100% seamless triangles without facet creasing.
    bool isHorizontal = abs(geomNormal.y) > 0.6;
    vec3 worldFaceNormal = isHorizontal ? vec3(0.0, 1.0, 0.0) : geomNormal;
    if (dot(mat3(u_view) * worldFaceNormal, viewDir) < 0.0) {
        worldFaceNormal = -worldFaceNormal;
    }

    // World tangent & bitangent: strictly aligned to global axes for planar surfaces
    // Analytic vertex wave slope matching main.vert displacement (eliminates triangle diagonal seams)
    uint packedMaterial = floatBitsToUint(Material);
    float vertAmp = float((packedMaterial >> 16u) & 0xFFu);
    float vertSlopeX = isHorizontal ? (0.05 * cos(WorldPos.x * 0.05 + t * 2.0) * vertAmp) : 0.0;
    float vertSlopeZ = isHorizontal ? (-0.05 * sin(WorldPos.z * 0.05 + t * 1.5) * vertAmp) : 0.0;
    vec2 totalSlope = waveDisp + vec2(vertSlopeX, vertSlopeZ);

    vec3 worldWaveNormal;
    if (isHorizontal) {
        worldWaveNormal = normalize(vec3(-totalSlope.x, worldFaceNormal.y, -totalSlope.y));
    } else {
        vec3 tAxis = abs(worldFaceNormal.y) > 0.9 ? vec3(1.0, 0.0, 0.0) : vec3(0.0, 1.0, 0.0);
        vec3 t1 = normalize(cross(tAxis, worldFaceNormal));
        vec3 t2 = cross(worldFaceNormal, t1);
        worldWaveNormal = normalize(worldFaceNormal + t1 * waveDisp.x + t2 * waveDisp.y);
    }
    vec3 waveNormal = normalize(mat3(u_view) * worldWaveNormal);
    vec3 V = viewDir;
    float NdotV = clamp(dot(waveNormal, V), 0.0, 1.0);
    // Glassy Fresnel: minimum base sheen (0.02 for water) rising to brilliant reflection at grazing angles
    float fresnel = 0.02 + 0.98 * pow(1.0 - NdotV, 5.0);
    // Sunlight and reflection vectors
    vec3 L = length(u_lightDir) > 0.1 ? normalize(u_lightDir) : normalize(vec3(0.35, 0.85, 0.40));
    vec3 H = normalize(L + V);
    float NdotH = max(dot(waveNormal, H), 0.0);
    vec3 R = reflect(-V, waveNormal);

    // Water Colors: lush aquatic tones blended with material texture if valid
    vec3 shallowColor = waterColorShallow;
    vec3 deepColor    = waterColorDeep;
    if (length(texColor) > 0.15) {
        // If texture has noticeable tint, incorporate it harmoniously
        shallowColor = mix(waterColorShallow, texColor * 1.25, 0.35);
        deepColor    = mix(waterColorDeep, texColor * 0.50, 0.40);
    }

    // Environment reflection (ambient sheen and sky gradient)
    vec3 ambientReflection = mix(vec3(0.20, 0.28, 0.32), shallowColor * 0.85, 0.40);
    vec3 skyReflection     = mix(envHorizon, envZenith, clamp(R.y * 0.5 + 0.5, 0.0, 1.0));
    vec3 envReflection     = mix(ambientReflection, skyReflection, 0.65);

    vec3 finalWater;

    if (isSolid) {
        // Impenetrable Water (Doom, Duke3D, Jedi floor liquids)
        vec2 refractedUV = TexCoords.xy + waveDisp * 0.015;
        vec3 bedTex = getDiffuse(vec3(refractedUV, TexCoords.z)).rgb;
        vec3 bedBase = (length(bedTex) > 0.03) ? bedTex : shallowColor;

        // Dynamic wave lighting reflecting the scene sunlight direction
        float waveLight = max(dot(waveNormal, L), 0.0) * 0.35 + 0.85;

        // Dancing caustics on the liquid surface
        float caustics = dancingSunlightCaustics(wPos, t);
        vec3 litWater = bedBase * waveLight + bedBase * (caustics * 0.35);

        // Harmonious aquatic presence without crushing the game's original texture
        vec3 waterBody = mix(litWater, shallowColor, 0.25);

        // Blend with environment reflection via Fresnel
        finalWater = mix(waterBody, envReflection, fresnel * 0.55 + 0.05);
    } else {
        // --- Penetrable / Swimmable Water (Quake swimmable volumes) ---
        vec2 distortion = waveNormal.xy * (isSea ? 0.0035 : 0.0020);
        vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);

        // Sample depth at distorted UV
        float sampleDepthRaw = texture(u_depthTex, distortedUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));

        // Edge rejection: if distorted ray hits foreground objects (e.g. columns or walls sticking out of water),
        // reject distortion to avoid smearing the column into the water
        if (sampleDepthRaw < 0.9999 && sampleDepth < waterDepth + 0.5) {
            distortedUV = screenUV;
            sampleDepthRaw = texture(u_depthTex, screenUV).r;
            sampleZN = sampleDepthRaw * 2.0 - 1.0;
            sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        }

        if (sampleDepthRaw < 0.9999 && sampleDepth > waterDepth + 0.1) {
            float depthDiff = max(sampleDepth - waterDepth, 0.0);
            vec3 refractionColor = refraction(depthDiff, distortion, distortedUV);

            // Water optical filter: transforms raw floor into rich aquatic tone
            float depthFactor = clamp(depthDiff * 0.020, 0.0, 1.0);
            vec3 bottomTinted = refractionColor * mix(vec3(0.35, 0.80, 0.92), vec3(0.12, 0.50, 0.70), depthFactor);

            // Underwater dancing caustics on submerged surfaces
            float caustics = dancingSunlightCaustics(wPos, t);
            float causticFade = clamp(1.0 - depthDiff * 0.015, 0.0, 1.0);
            bottomTinted += bottomTinted * caustics * 0.25 * causticFade;

            // Optical depth extinction: light is absorbed exponentially (Beer-Lambert law)
            // Shallow water is transparent; deeper water smoothly obscures the bottom
            float turbidity = 1.0 - exp(-depthDiff * 0.022);
            vec3 waterVolumeColor = mix(shallowColor, deepColor, depthFactor);

            // Submerged scene: blend from clear refracted bottom into atmospheric liquid volume
            vec3 waterBody = mix(bottomTinted, waterVolumeColor, turbidity);
            waterBody = max(waterBody, shallowColor * 0.15);

            // Subsurface Scattering light glow through waves facing sun
            float sssSun = pow(clamp(dot(V, -L), 0.0, 1.0), 3.0);
            waterBody += shallowColor * (sssSun * 0.30);

            // Shoreline / Contact soft edge foam
            float shoreFoam = clamp(1.0 - depthDiff / 6.0, 0.0, 1.0);
            shoreFoam = pow(shoreFoam, 2.0) * (0.45 + 0.45 * sin(t * 3.0 + (wPos.x + wPos.y) * 0.15));

            // Surface reflection blend via Fresnel (gives glassy liquid presence)
            finalWater = mix(waterBody, envReflection, fresnel * 0.75 + 0.04);

            // Apply soft shoreline foam
            finalWater = mix(finalWater, vec3(0.92, 0.96, 1.0), shoreFoam * 0.35);
        } else {
            // Surface fallback
            vec3 inscatterColor = mix(shallowColor, deepColor, 0.5);
            finalWater = mix(inscatterColor, envReflection, fresnel * 0.75 + 0.04);
        }
    }

    // Specular sunlight glints (Blinn-Phong sun path)
    float specBroad = pow(NdotH, 32.0) * 0.30;
    float specCore  = pow(NdotH, 160.0) * 1.80;
    vec3 sunSpecular = vec3(1.0, 0.98, 0.92) * (specBroad + specCore) * (0.25 + 0.75 * fresnel);
    finalWater += sunSpecular;

    FragColor = vec4(finalWater, 1.0);
    // Write specular glints to bloom channel
    float bloomLum = dot(sunSpecular, vec3(0.2126, 0.7152, 0.0722));
    vec3 bloomColor = bloomLum > 0.8 ? (sunSpecular - 0.8) * 1.2 : vec3(0.0);
    BrightColor = vec4(bloomColor, 1.0);
}

void main() {
    uint packedMaterial = floatBitsToUint(Material);
    uint liquidType     = (packedMaterial)        & 0xFFu;
    uint solid          = (packedMaterial >>  8u) & 0xFFu;
    bool isSolid = (solid != 0u);

    if (liquidType == LIQUID_WATER) {
        renderWater(isSolid, false);
    } else if (liquidType == LIQUID_SEA) {
        renderWater(isSolid, true);
    } else if (liquidType == LIQUID_LAVA) {
        renderLava(isSolid);
    } else if (liquidType == LIQUID_SLIME) {
        renderSlime(isSolid);
    } else if (liquidType == LIQUID_BLOOD) {
        renderBlood(isSolid);
    } else if (liquidType == LIQUID_FOG) {
        renderFog(isSolid);
    } else {
        renderWater(isSolid, false);
    }
}
