#version 330 core
out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D u_ssaoInput;
uniform sampler2D u_depthTex;
uniform vec2 u_texelSize;  // 1.0 / halfResSize
uniform vec4 u_projParams; // (1/P[0], 1/P[5], P[10], P[14])

float linearizeViewZ(float depth) {
    float zNdc = depth * 2.0 - 1.0;
    return -u_projParams.w / (u_projParams.z + zNdc);
}

void accumBilateral(vec2 offsetPx, float zC, float invSigma2, float spatialW, inout float sumAO, inout float sumW) {
    vec2 uv = clamp(TexCoords + offsetPx * u_texelSize, 0.001, 0.999);
    float d = texture(u_depthTex, uv).r;
    if (d >= 0.9999) {
        return;
    }
    float zS = linearizeViewZ(d);
    float dz = zS - zC;
    float w = spatialW * exp(-(dz * dz) * invSigma2);
    float ao = texture(u_ssaoInput, uv).r;
    sumAO += ao * w;
    sumW += w;
}

void main() {
    float dC = texture(u_depthTex, TexCoords).r;
    if (dC >= 0.9999) {
        FragColor = vec4(1.0);
        return;
    }

    float zC = linearizeViewZ(dC);
    float rangeSigma = max(3.5, abs(zC) * 0.04);
    float invSigma2 = 0.5 / (rangeSigma * rangeSigma);

    float sumAO = texture(u_ssaoInput, TexCoords).r;
    float sumW = 1.0;

    // Symmetrical 3x3 core + 4 outer axial taps for artifact-free bilateral smoothing.
    accumBilateral(vec2(-1.0, -1.0), zC, invSigma2, 0.70, sumAO, sumW);
    accumBilateral(vec2( 0.0, -1.0), zC, invSigma2, 0.85, sumAO, sumW);
    accumBilateral(vec2( 1.0, -1.0), zC, invSigma2, 0.70, sumAO, sumW);
    accumBilateral(vec2(-1.0,  0.0), zC, invSigma2, 0.85, sumAO, sumW);
    accumBilateral(vec2( 1.0,  0.0), zC, invSigma2, 0.85, sumAO, sumW);
    accumBilateral(vec2(-1.0,  1.0), zC, invSigma2, 0.70, sumAO, sumW);
    accumBilateral(vec2( 0.0,  1.0), zC, invSigma2, 0.85, sumAO, sumW);
    accumBilateral(vec2( 1.0,  1.0), zC, invSigma2, 0.70, sumAO, sumW);

    accumBilateral(vec2(-2.0,  0.0), zC, invSigma2, 0.45, sumAO, sumW);
    accumBilateral(vec2( 2.0,  0.0), zC, invSigma2, 0.45, sumAO, sumW);
    accumBilateral(vec2( 0.0, -2.0), zC, invSigma2, 0.45, sumAO, sumW);
    accumBilateral(vec2( 0.0,  2.0), zC, invSigma2, 0.45, sumAO, sumW);

    float ao = clamp(sumAO / max(sumW, 1e-4), 0.0, 1.0);
    FragColor = vec4(ao, ao, ao, 1.0);
}
