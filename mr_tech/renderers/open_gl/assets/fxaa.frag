#version 330 core
out vec4 FragColor;
in vec2 TexCoords;

uniform sampler2D u_screenTex;
uniform vec2 u_texelSize;

// Perceptual luminance (Rec. 601) used for FXAA edge contrast detection.
float fxaaLuma(vec3 rgb) {
    return dot(rgb, vec3(0.299, 0.587, 0.114));
}

void main() {
    const float EDGE_THRESHOLD_MIN = 0.0312;
    const float EDGE_THRESHOLD_MAX = 0.125;
    const float SUBPIX_QUALITY = 0.75;
    const int SEARCH_STEPS = 12;

    vec3 rgbM = texture(u_screenTex, TexCoords).rgb;
    float lumaM = fxaaLuma(rgbM);

    float lumaD = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2(0, -1)).rgb);
    float lumaU = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2(0,  1)).rgb);
    float lumaL = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2(-1, 0)).rgb);
    float lumaR = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2( 1, 0)).rgb);

    float lumaMin = min(lumaM, min(min(lumaD, lumaU), min(lumaL, lumaR)));
    float lumaMax = max(lumaM, max(max(lumaD, lumaU), max(lumaL, lumaR)));
    float lumaRange = lumaMax - lumaMin;

    if (lumaRange < max(EDGE_THRESHOLD_MIN, lumaMax * EDGE_THRESHOLD_MAX)) {
        FragColor = vec4(rgbM, 1.0);
        return;
    }

    float lumaDL = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2(-1, -1)).rgb);
    float lumaUR = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2( 1,  1)).rgb);
    float lumaUL = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2(-1,  1)).rgb);
    float lumaDR = fxaaLuma(textureOffset(u_screenTex, TexCoords, ivec2( 1, -1)).rgb);

    float lumaDU = lumaD + lumaU;
    float lumaLR = lumaL + lumaR;

    float lumaLeftCorners  = lumaDL + lumaUL;
    float lumaDownCorners  = lumaDL + lumaDR;
    float lumaRightCorners = lumaDR + lumaUR;
    float lumaUpCorners    = lumaUR + lumaUL;

    float edgeHorz = abs(-2.0 * lumaL + lumaLeftCorners) + abs(-2.0 * lumaM + lumaDU) * 2.0 + abs(-2.0 * lumaR + lumaRightCorners);
    float edgeVert = abs(-2.0 * lumaU + lumaUpCorners)   + abs(-2.0 * lumaM + lumaLR) * 2.0 + abs(-2.0 * lumaD + lumaDownCorners);
    bool isHorizontal = (edgeHorz >= edgeVert);

    float luma1 = isHorizontal ? lumaD : lumaL;
    float luma2 = isHorizontal ? lumaU : lumaR;
    float gradient1 = luma1 - lumaM;
    float gradient2 = luma2 - lumaM;
    bool is1Steepest = abs(gradient1) >= abs(gradient2);
    float gradientScaled = 0.25 * max(abs(gradient1), abs(gradient2));

    float stepLength = isHorizontal ? u_texelSize.y : u_texelSize.x;
    float lumaLocalAverage = 0.0;
    if (is1Steepest) {
        stepLength = -stepLength;
        lumaLocalAverage = 0.5 * (luma1 + lumaM);
    } else {
        lumaLocalAverage = 0.5 * (luma2 + lumaM);
    }

    vec2 currentUv = TexCoords;
    if (isHorizontal) {
        currentUv.y += stepLength * 0.5;
    } else {
        currentUv.x += stepLength * 0.5;
    }

    vec2 offset = isHorizontal ? vec2(u_texelSize.x, 0.0) : vec2(0.0, u_texelSize.y);
    vec2 uv1 = currentUv - offset;
    vec2 uv2 = currentUv + offset;

    float lumaEnd1 = fxaaLuma(texture(u_screenTex, uv1).rgb) - lumaLocalAverage;
    float lumaEnd2 = fxaaLuma(texture(u_screenTex, uv2).rgb) - lumaLocalAverage;
    bool reached1 = abs(lumaEnd1) >= gradientScaled;
    bool reached2 = abs(lumaEnd2) >= gradientScaled;
    bool reachedBoth = reached1 && reached2;

    if (!reached1) uv1 -= offset;
    if (!reached2) uv2 += offset;

    if (!reachedBoth) {
        for (int i = 2; i < SEARCH_STEPS; i++) {
            if (!reached1) lumaEnd1 = fxaaLuma(texture(u_screenTex, uv1).rgb) - lumaLocalAverage;
            if (!reached2) lumaEnd2 = fxaaLuma(texture(u_screenTex, uv2).rgb) - lumaLocalAverage;
            reached1 = abs(lumaEnd1) >= gradientScaled;
            reached2 = abs(lumaEnd2) >= gradientScaled;
            reachedBoth = reached1 && reached2;
            if (!reached1) uv1 -= offset * 1.5;
            if (!reached2) uv2 += offset * 1.5;
            if (reachedBoth) break;
        }
    }

    float distance1 = isHorizontal ? (TexCoords.x - uv1.x) : (TexCoords.y - uv1.y);
    float distance2 = isHorizontal ? (uv2.x - TexCoords.x) : (uv2.y - TexCoords.y);
    bool isDirection1 = distance1 < distance2;
    float distanceFinal = min(distance1, distance2);
    float edgeThickness = distance1 + distance2;

    float pixelOffset = -distanceFinal / edgeThickness + 0.5;
    bool isLumaMSmaller = lumaM < lumaLocalAverage;
    bool correctVariation = ((isDirection1 ? lumaEnd1 : lumaEnd2) < 0.0) != isLumaMSmaller;
    float finalOffset = correctVariation ? pixelOffset : 0.0;

    float lumaAverage = (1.0 / 12.0) * (2.0 * (lumaDU + lumaLR) + lumaLeftCorners + lumaRightCorners);
    float subPixelOffset1 = clamp(abs(lumaAverage - lumaM) / lumaRange, 0.0, 1.0);
    float subPixelOffset2 = (-2.0 * subPixelOffset1 + 3.0) * subPixelOffset1 * subPixelOffset1;
    float subPixelOffsetFinal = subPixelOffset2 * subPixelOffset2 * SUBPIX_QUALITY;

    finalOffset = max(finalOffset, subPixelOffsetFinal);

    vec2 finalUv = TexCoords;
    if (isHorizontal) {
        finalUv.y += finalOffset * stepLength;
    } else {
        finalUv.x += finalOffset * stepLength;
    }

    FragColor = vec4(texture(u_screenTex, finalUv).rgb, 1.0);
}
