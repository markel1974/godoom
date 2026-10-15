#version 330 core

in vec3 TexCoords;
in vec3 ViewPos;
in vec3 WorldPos;

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
uniform int u_liquidType;

const int LIQUID_WATER = 0; // Acqua limpida trasparente (default)
const int LIQUID_SEA   = 1; // Mare con onde ampie e schiuma su creste e riva
const int LIQUID_LAVA  = 2; // Lava incandescente opaca con vene di magma
const int LIQUID_SLIME = 3; // Slime / Nukage acido radioattivo
const int LIQUID_BLOOD = 4; // Sangue rosso denso
const int LIQUID_FOG   = 5; // Nebbia volumetrica atmosferica

// Water / Liquid diffuse texture sampler
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

void main() {
    // Diffuse texture from material
    vec3 texColor = getDiffuse(TexCoords).rgb;

    // Screen-space coordinates & linear view depths
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    float waterDepth = abs(ViewPos.z);
    float t = u_time;

    // Use continuous WorldPos.xz to prevent texture-tiling repetition across sectors
    vec2 wPos = WorldPos.xz;

    // -------------------------------------------------------------------------
    // LIQUID TYPE 5: FOG (Nebbia volumetrica)
    // -------------------------------------------------------------------------
    if (u_liquidType == LIQUID_FOG) {
        float sampleDepthRaw = texture(u_depthTex, screenUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        float depthDiff = max(sampleDepth - waterDepth, 0.0);
        if (sampleDepthRaw >= 0.9999) depthDiff = max(depthDiff, 150.0);

        float fogFactor = clamp(1.0 - exp(-depthDiff * 0.015), 0.0, 1.0);
        vec3 bg = texture(u_refractionTex, screenUV).rgb;
        vec3 fogColor = mix(texColor, vec3(0.72, 0.76, 0.82), 0.35);
        vec3 finalFog = mix(bg, fogColor, fogFactor * 0.92);

        FragColor = vec4(finalFog, 1.0);
        BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }

    // -------------------------------------------------------------------------
    // LIQUID TYPE 2: LAVA (Lava incandescente opaca)
    // -------------------------------------------------------------------------
    if (u_liquidType == LIQUID_LAVA) {
        float lavaT = t * 0.5;
        float n1 = fbm(wPos * 0.015 + vec2(lavaT * 0.04, -lavaT * 0.03));
        float n2 = fbm(wPos * 0.035 + vec2(-lavaT * 0.05, lavaT * 0.04));
        float lavaNoise = mix(n1, n2, 0.5);

        vec3 lavaCrust = texColor * 0.35;
        vec3 lavaHot   = vec3(1.35, 0.38, 0.02);
        vec3 lavaCore  = vec3(1.90, 1.25, 0.30);

        float tear = clamp(lavaNoise * 1.5 - 0.25, 0.0, 1.0);
        vec3 finalLava = mix(lavaCrust, lavaHot, smoothstep(0.20, 0.60, tear));
        finalLava = mix(finalLava, lavaCore, smoothstep(0.68, 0.95, tear));

        float sampleDepthRaw = texture(u_depthTex, screenUV).r;
        float sampleZN = sampleDepthRaw * 2.0 - 1.0;
        float sampleDepth = (2.0 * u_near * u_far) / (u_far + u_near - sampleZN * (u_far - u_near));
        float depthDiff = max(sampleDepth - waterDepth, 0.0);

        const float LAVA_RIM = 5.0;
        if (depthDiff < LAVA_RIM) {
            float rimGrad = 1.0 - (depthDiff / LAVA_RIM);
            float rimNoise = fbm(wPos * 0.06 + vec2(lavaT * 0.15, -lavaT * 0.12));
            float sizzle = smoothstep(0.35, 0.75, rimGrad + rimNoise * 0.4);
            finalLava = mix(finalLava, vec3(1.95, 0.95, 0.18), sizzle * 0.85);
        }

        float glow = dot(finalLava, vec3(0.35, 0.55, 0.10));
        vec3 bloomLava = glow > 0.85 ? (finalLava - 0.65) * 1.3 : vec3(0.0);

        FragColor = vec4(finalLava, 1.0);
        BrightColor = vec4(bloomLava, 1.0);
        return;
    }

    // -------------------------------------------------------------------------
    // WAVE SIMULATION (WATER, SEA, SLIME, BLOOD)
    // -------------------------------------------------------------------------
    vec2 distortion = vec2(0.0);
    float waveElevation = 0.0;

    if (u_liquidType == LIQUID_SEA) {
        // Sea / Ocean: larger rolling swells based on continuous WorldPos
        vec2 p = wPos * 0.012;
        vec2 sw1 = vec2(sin(p.x * 1.5 + p.y * 0.7 + t * 0.9), cos(p.y * 1.4 - p.x * 0.8 + t * 0.8));
        vec2 sw2 = vec2(sin(p.y * 2.2 + p.x * 1.1 - t * 0.7), cos(p.x * 1.9 - p.y * 1.2 - t * 0.85));
        float seaNoise = fbm(p * 0.4 + vec2(t * 0.03, -t * 0.02));
        distortion = (sw1 * 0.65 + sw2 * 0.35) * (0.0035 + seaNoise * 0.0020);
        waveElevation = (sw1.x + sw2.y) * 0.5 + seaNoise * 0.5;
    } else {
        // Standard Water / Slime / Blood: gentle, fluid, organic ripples
        vec2 p = wPos * 0.025;
        vec2 rw1 = vec2(sin(p.x * 1.7 + p.y * 0.8 + t * 1.2), cos(p.y * 1.5 - p.x * 0.9 + t * 1.0));
        vec2 rw2 = vec2(sin(p.y * 2.3 + p.x * 1.3 - t * 0.85), cos(p.x * 2.1 - p.y * 1.4 - t * 1.15));
        vec2 rw3 = vec2(sin((p.x + p.y) * 3.1 + t * 0.65), cos((p.x - p.y) * 2.8 - t * 0.75));
        float ripNoise = fbm(p * 0.6 + vec2(t * 0.03, -t * 0.025));
        distortion = (rw1 * 0.50 + rw2 * 0.35 + rw3 * 0.15) * (0.0022 + ripNoise * 0.0012);
        waveElevation = (rw1.x + rw2.y) * 0.5;
    }

    // Refraction sampling with foreground depth rejection
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

    float depthDiff = max(sampleDepth - waterDepth, 0.0);
    if (sampleDepthRaw >= 0.9999) {
        depthDiff = max(depthDiff, 150.0);
    }

    // -------------------------------------------------------------------------
    // MULTI-TAP OPTICAL DIFFUSION & CHROMATIC DISPERSION
    // Submerged elements become softer and less razor-sharp with increasing depth
    // -------------------------------------------------------------------------
    float blurRadius = clamp(depthDiff * 0.00007, 0.0007, 0.0035);
    vec2 dispDir = length(distortion) > 1e-5 ? normalize(distortion) : vec2(0.707, 0.707);
    vec2 dispOffset = dispDir * (blurRadius * 0.85);

    // 5-tap kernel around distorted coordinate
    vec3 cCenter = texture(u_refractionTex, distortedUV).rgb;
    vec3 c1 = texture(u_refractionTex, distortedUV + vec2(blurRadius, blurRadius * 0.55)).rgb;
    vec3 c2 = texture(u_refractionTex, distortedUV - vec2(blurRadius, blurRadius * 0.55)).rgb;
    vec3 c3 = texture(u_refractionTex, distortedUV + vec2(-blurRadius * 0.55, blurRadius)).rgb;
    vec3 c4 = texture(u_refractionTex, distortedUV + vec2(blurRadius * 0.55, -blurRadius)).rgb;
    vec3 diffusedRefraction = cCenter * 0.36 + (c1 + c2 + c3 + c4) * 0.16;

    // Chromatic dispersion (splitting red and blue slightly along refraction wave gradient)
    float rCh = texture(u_refractionTex, distortedUV + dispOffset).r;
    float bCh = texture(u_refractionTex, distortedUV - dispOffset).b;
    vec3 refractionColor = vec3(rCh * 0.45 + diffusedRefraction.r * 0.55,
                                diffusedRefraction.g,
                                bCh * 0.45 + diffusedRefraction.b * 0.55);

    // Surface-aligned normal perturbation
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

    // -------------------------------------------------------------------------
    // LIQUID TYPE 3: SLIME / NUKAGE (Acido radioattivo)
    // -------------------------------------------------------------------------
    if (u_liquidType == LIQUID_SLIME) {
        const vec3 SLIME_EXTINCTION = vec3(0.055, 0.010, 0.050);
        vec3 transmittance = clamp(exp(-depthDiff * SLIME_EXTINCTION), 0.0, 1.0);
        vec3 transmittedBg = refractionColor * transmittance;

        vec3 shallowSlime = mix(vec3(0.18, 0.78, 0.12), texColor * 1.15, 0.50);
        vec3 deepSlime    = mix(vec3(0.03, 0.22, 0.02), texColor * 0.35, 0.60);
        float depthFactor = clamp(1.0 - exp(-depthDiff * 0.020), 0.0, 1.0);
        vec3 slimeFog = mix(shallowSlime, deepSlime, depthFactor);

        float turbidity = mix(0.35, 0.88, depthFactor);
        vec3 slimeBody = mix(transmittedBg, slimeFog, turbidity);

        float surfaceOpacity = mix(0.25, 0.65, fresnel);
        vec3 finalSlime = mix(slimeBody, texColor, surfaceOpacity);

        // Shoreline bubbling scum
        const float SLIME_FOAM = 5.0;
        if (depthDiff < SLIME_FOAM) {
            float scumGrad = 1.0 - (depthDiff / SLIME_FOAM);
            float scumNoise = fbm(wPos * 0.08 + vec2(t * 0.12, -t * 0.08));
            float scumFactor = smoothstep(0.35, 0.75, scumGrad * 1.2 + scumNoise * 0.4 - 0.2);
            finalSlime = mix(finalSlime, vec3(0.65, 0.95, 0.35), scumFactor * 0.60);
        }

        float spec = pow(NdotH, 32.0) * 0.40 * (0.30 + 0.70 * fresnel);
        finalSlime += vec3(0.8, 1.0, 0.6) * spec;

        float slimeGlow = finalSlime.g;
        vec3 bloomSlime = slimeGlow > 0.75 ? vec3(0.12, 0.55, 0.08) * (slimeGlow - 0.75) * 2.0 : vec3(0.0);

        FragColor = vec4(finalSlime, 1.0);
        BrightColor = vec4(bloomSlime, 1.0);
        return;
    }

    // -------------------------------------------------------------------------
    // LIQUID TYPE 4: BLOOD (Sangue viscoso)
    // -------------------------------------------------------------------------
    if (u_liquidType == LIQUID_BLOOD) {
        const vec3 BLOOD_EXTINCTION = vec3(0.010, 0.055, 0.055);
        vec3 transmittance = clamp(exp(-depthDiff * BLOOD_EXTINCTION), 0.0, 1.0);
        vec3 transmittedBg = refractionColor * transmittance;

        vec3 shallowBlood = mix(vec3(0.55, 0.05, 0.03), texColor * 1.10, 0.45);
        vec3 deepBlood    = mix(vec3(0.12, 0.01, 0.01), texColor * 0.30, 0.60);
        float depthFactor = clamp(1.0 - exp(-depthDiff * 0.020), 0.0, 1.0);
        vec3 bloodFog = mix(shallowBlood, deepBlood, depthFactor);

        float turbidity = mix(0.40, 0.90, depthFactor);
        vec3 bloodBody = mix(transmittedBg, bloodFog, turbidity);

        float surfaceOpacity = mix(0.25, 0.70, fresnel);
        vec3 finalBlood = mix(bloodBody, texColor, surfaceOpacity);

        float spec = pow(NdotH, 48.0) * 0.50 * (0.35 + 0.65 * fresnel);
        finalBlood += vec3(1.0, 0.85, 0.85) * spec;

        FragColor = vec4(finalBlood, 1.0);
        BrightColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }

    // -------------------------------------------------------------------------
    // LIQUID TYPE 1: SEA / OCEAN (Mare con onde ampie e schiuma)
    // -------------------------------------------------------------------------
    if (u_liquidType == LIQUID_SEA) {
        // Deep marine oceanic extinction
        vec3 seaTint = mix(vec3(0.04, 0.40, 0.55), texColor * 1.1, 0.30);
        float depthFactor = clamp(depthDiff * 0.015, 0.0, 1.0);
        vec3 transmitted = mix(refractionColor, refractionColor * seaTint * 1.6, depthFactor * 0.60);

        vec3 deepAbyss = mix(vec3(0.01, 0.10, 0.22), texColor * 0.4, 0.45);
        vec3 seaBody = mix(transmitted, deepAbyss, clamp(depthDiff * 0.008 - 0.1, 0.0, 0.60));

        float surfaceMix = mix(0.10, 0.45, fresnel);
        vec3 finalSea = mix(seaBody, texColor, surfaceMix);

        // 1. Wave Crest Foam (Schiuma sulle creste delle onde)
        float crestNoise = fbm(wPos * 0.025 + vec2(t * 0.08, -t * 0.06));
        float crestFoam = smoothstep(0.35, 0.75, waveElevation * 0.9 + crestNoise * 0.45 - 0.20);
        vec3 foamWhite = vec3(0.95, 0.98, 1.0);
        finalSea = mix(finalSea, foamWhite, crestFoam * 0.65);

        // 2. Shoreline Surf Foam (Schiuma a riva contro pareti e coste)
        const float SHORE_DISTANCE = 8.0;
        if (depthDiff < SHORE_DISTANCE) {
            float shoreGrad = 1.0 - (depthDiff / SHORE_DISTANCE);
            float shoreNoise = fbm(wPos * 0.05 + vec2(t * 0.14, -t * 0.10));
            float shoreFoam = smoothstep(0.30, 0.70, shoreGrad * 1.3 + shoreNoise * 0.45 - 0.25);
            finalSea = mix(finalSea, foamWhite, shoreFoam * 0.75);
        }

        // Specular highlight
        float spec = pow(NdotH, 48.0) * 0.55 * (0.25 + 0.75 * fresnel);
        finalSea += vec3(1.0, 0.98, 0.94) * spec;

        FragColor = vec4(finalSea, 1.0);
        float brightness = dot(finalSea, vec3(0.2126, 0.7152, 0.0722));
        BrightColor = vec4(brightness > 1.3 ? finalSea * 0.5 : vec3(0.0), 1.0);
        return;
    }

    // -------------------------------------------------------------------------
    // LIQUID TYPE 0: WATER (Acqua default: trasparente, ma elementi subacquei ammorbiditi)
    // -------------------------------------------------------------------------
    // Water tint filter
    vec3 waterTint = mix(vec3(0.06, 0.45, 0.58), texColor * 1.15, 0.30);
    float depthFactor = clamp(depthDiff * 0.015, 0.0, 1.0);

    // Contrast attenuation: underwater forward scattering softens sharp edges and harsh black shadows
    float lumBg = dot(refractionColor, vec3(0.299, 0.587, 0.114));
    vec3 softenedBg = mix(refractionColor, waterTint * lumBg * 1.6, depthFactor * 0.32);

    // Background transmission: filtered by aquatic color
    vec3 transmitted = mix(softenedBg, softenedBg * waterTint * 1.7, depthFactor * 0.50);

    // Deep water gradual transition (only in deep water, never muddy)
    vec3 deepAbyss = mix(vec3(0.015, 0.12, 0.22), texColor * 0.40, 0.40);
    vec3 waterBody = mix(transmitted, deepAbyss, clamp(depthDiff * 0.007 - 0.15, 0.0, 0.50));

    // Surface reflection: light at normal angles (~7%), reflective at glancing angles (Fresnel)
    float surfaceMix = mix(0.07, 0.42, fresnel);
    vec3 finalWater = mix(waterBody, texColor, surfaceMix);

    // Specular sunlight glints
    float spec = pow(NdotH, 48.0) * 0.55 * (0.25 + 0.75 * fresnel);
    finalWater += vec3(1.0, 0.98, 0.94) * spec;

    // Soft geometry edge fade: avoids harsh pixelated polygon edge against walls
    float edgeFade = clamp(depthDiff * 0.8, 0.0, 1.0);
    finalWater = mix(refractionColor, finalWater, edgeFade);

    // Output
    FragColor = vec4(finalWater, 1.0);

    // Bloom emission for sun glints
    float brightness = dot(finalWater, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(brightness > 1.3 ? finalWater * 0.5 : vec3(0.0), 1.0);
}