#version 330 core

in vec3 TexCoords;
in vec3 ViewPos;

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_texture[4];

uniform sampler2D u_refractionTex;
uniform sampler2D u_depthTex;

uniform vec2 u_resolution;
uniform float u_time;

uniform float u_near;
uniform float u_far;


// Water texture
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

// Hash
float hash21(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}

// Value noise
float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    // Smooth interpolation
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
    const float WATER_FOG_R = 0.0;
    const float WATER_FOG_G = 0.2;
    const float WATER_FOG_B = 0.3;
    const float WATER_ABSORPTION = 0.02;
    // Overall temporal speed. Lower = slower water movement.
    const float WATER_TIME_SCALE = 0.4;
    // Temporal frequencies.
    const float WAVE1_SPEED_X = 1.31;
    const float WAVE1_SPEED_Y = 1.07;
    const float WAVE2_SPEED_X = 0.83;
    const float WAVE2_SPEED_Y = 1.17;
    const float WAVE3_SPEED_X = 0.61;
    const float WAVE3_SPEED_Y = 0.73;
    // Large-scale noise.
    const float NOISE1_SCALE = 1.8;
    const float NOISE1_SPEED_X = 0.07;
    const float NOISE1_SPEED_Y = -0.045;
    // Small-scale noise.
    const float NOISE2_SCALE = 3.7;
    const float NOISE2_SPEED_X = -0.11;
    const float NOISE2_SPEED_Y = 0.065;
    // Local amplitude variation.
    const float VARIATION_MIN = 0.65;
    const float VARIATION_MAX = 1.25;
    // Wave 1 spatial frequencies.
    const float WAVE1_X = 23.0;
    const float WAVE1_XY = 7.0;
    const float WAVE1_Y = 27.0;
    const float WAVE1_YX = 5.0;
    // Wave 2 spatial frequencies.
    const float WAVE2_Y = 37.0;
    const float WAVE2_XY = 11.0;
    const float WAVE2_X = 31.0;
    const float WAVE2_YX = 9.0;
    // Wave 3 spatial frequencies.
    const float WAVE3_SUM = 52.0;
    const float WAVE3_DIFF = 47.0;
    // Relative contribution of each wave family.
    const float WAVE1_BASE = 0.45;
    const float WAVE1_NOISE = 0.75;
    const float WAVE2_BASE = 0.25;
    const float WAVE2_NOISE = 0.65;
    const float WAVE3_BASE = 0.10;
    const float WAVE3_NOISE = 0.45;
    // Screen-space distortion strength.
    const float DISTORTION_BASE = 0.0015;
    const float DISTORTION_LUM = 0.0055;

    vec2 screenUV = gl_FragCoord.xy / u_resolution;

    // Water texture
    vec3 texColor = getDiffuse(TexCoords).rgb;
    float lum = dot(texColor, vec3(0.299, 0.587, 0.114));

    // TIME
    float t = u_time * WATER_TIME_SCALE;
    // NOISE
    float n1 = fbm(TexCoords.xy * NOISE1_SCALE + vec2(t * NOISE1_SPEED_X, t * NOISE1_SPEED_Y));
    float n2 = fbm(TexCoords.xy * NOISE2_SCALE + vec2(t * NOISE2_SPEED_X, t * NOISE2_SPEED_Y));
    // LOCAL AMPLITUDE VARIATION
    float variation = mix(VARIATION_MIN, VARIATION_MAX, n1);
    // WAVE 1
    vec2 wave1 = vec2(sin(TexCoords.x * WAVE1_X + TexCoords.y * WAVE1_XY + t * WAVE1_SPEED_X), cos(TexCoords.y * WAVE1_Y - TexCoords.x * WAVE1_YX + t * WAVE1_SPEED_Y));
    // WAVE 2
    vec2 wave2 = vec2(sin(TexCoords.y * WAVE2_Y + TexCoords.x * WAVE2_XY - t * WAVE2_SPEED_X), cos(TexCoords.x * WAVE2_X - TexCoords.y * WAVE2_YX - t * WAVE2_SPEED_Y));
    // WAVE 3
    vec2 wave3 = vec2(sin((TexCoords.x + TexCoords.y) * WAVE3_SUM + t * WAVE3_SPEED_X), cos((TexCoords.x - TexCoords.y) * WAVE3_DIFF - t * WAVE3_SPEED_Y));
    // COMBINE WAVES
    vec2 distortion = wave1 * (WAVE1_BASE + n1 * WAVE1_NOISE) + wave2 * (WAVE2_BASE + n2 * WAVE2_NOISE) + wave3 * (WAVE3_BASE + n1 * n2 * WAVE3_NOISE);

    // DISTORTION STRENGTH
    distortion *= (DISTORTION_BASE + lum * DISTORTION_LUM);
    distortion *= variation;

    // REFRACTION UV
    vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);
    // Refraction
    vec3 refractionColor = texture(u_refractionTex, distortedUV).rgb;

    // Background depth
    float bgDepthRaw = texture(u_depthTex, screenUV).r;
    float zN = bgDepthRaw * 2.0 - 1.0;
    float bgDepth = (2.0 * u_near * u_far) / (u_far + u_near - zN * (u_far - u_near));

    // Water surface depth
    float waterDepth = abs(ViewPos.z);
    float depthDiff = max(bgDepth - waterDepth, 0.0);

    // Beer-Lambert absorption
    vec3 waterFogColor = vec3(WATER_FOG_R, WATER_FOG_G, WATER_FOG_B);
    float transmittance = exp(-depthDiff * WATER_ABSORPTION);
    vec3 finalColor = refractionColor * transmittance + waterFogColor * (1.0 - transmittance);

    // Mix water texture
    finalColor = mix(finalColor, finalColor * texColor, 0.35);

    // Output
    FragColor = vec4(finalColor, 0.85);
    float brightness = dot(finalColor, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(brightness > 3.0? finalColor: vec3(0.0), 1.0);
}