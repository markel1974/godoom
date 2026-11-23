#version 330 core
out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D u_depthTex;
uniform vec4 u_projParams; // (1/P[0], 1/P[5], P[10], P[14])
uniform vec2 u_texelSize;  // 1.0 / fullResSize
uniform float u_radius;
uniform float u_bias;

const float PI = 3.14159265359;

float interleavedGradientNoise(vec2 px) {
    vec3 magic = vec3(0.06711056, 0.00583715, 52.9829189);
    return fract(magic.z * fract(dot(px, magic.xy)));
}

vec3 reconstructViewPos(vec2 uv, float depth) {
    float zNdc = depth * 2.0 - 1.0;
    float zv = -u_projParams.w / (u_projParams.z + zNdc);
    vec2 ndc = uv * 2.0 - 1.0;
    return vec3(ndc.x * (-zv) * u_projParams.x, ndc.y * (-zv) * u_projParams.y, zv);
}

vec3 reconstructBestFitNormal(vec2 uv, vec3 P, float depthC) {
    vec2 dx = vec2(u_texelSize.x, 0.0);
    vec2 dy = vec2(0.0, u_texelSize.y);

    float dL = texture(u_depthTex, clamp(uv - dx, 0.001, 0.999)).r;
    float dR = texture(u_depthTex, clamp(uv + dx, 0.001, 0.999)).r;
    float dD = texture(u_depthTex, clamp(uv - dy, 0.001, 0.999)).r;
    float dU = texture(u_depthTex, clamp(uv + dy, 0.001, 0.999)).r;

    vec3 dPdx = (abs(dR - depthC) < abs(depthC - dL))
        ? (reconstructViewPos(uv + dx, dR) - P)
        : (P - reconstructViewPos(uv - dx, dL));
    vec3 dPdy = (abs(dU - depthC) < abs(depthC - dD))
        ? (reconstructViewPos(uv + dy, dU) - P)
        : (P - reconstructViewPos(uv - dy, dD));

    vec3 n = cross(dPdx, dPdy);
    float lenSq = dot(n, n);
    if (lenSq <= 1e-10) {
        return vec3(0.0, 0.0, 1.0);
    }
    n *= inversesqrt(lenSq);
    if (dot(n, -P) < 0.0) {
        n = -n;
    }
    return n;
}

void main() {
    float depth = texture(u_depthTex, TexCoords).r;
    if (depth >= 0.9999) {
        FragColor = vec4(1.0);
        return;
    }

    vec3 P = reconstructViewPos(TexCoords, depth);
    vec3 N = reconstructBestFitNormal(TexCoords, P, depth);

    float viewDist = max(-P.z, 0.5);
    float radius = max(u_radius, 4.0);
    float rSq = radius * radius;

    // Project world radius to UV span while keeping cache-friendly screen bounds.
    float projUvX = (radius * 0.5) / max(viewDist * u_projParams.x, 0.001);
    float projUvY = (radius * 0.5) / max(viewDist * u_projParams.y, 0.001);
    vec2 maxScreenUV = vec2(min(projUvX, 0.055), min(projUvY, 0.055));
    vec2 minScreenUV = u_texelSize * 4.0;
    vec2 spanUV = max(maxScreenUV, minScreenUV);

    float ign = interleavedGradientNoise(gl_FragCoord.xy);
    float jitterFrac = fract(ign * 7.31);
    float baseAngle = ign * 2.0 * PI;

    const int NUM_DIRS = 6;
    const int NUM_STEPS = 3;
    float occSum = 0.0;

    for (int d = 0; d < NUM_DIRS; ++d) {
        float theta = baseAngle + float(d) * (2.0 * PI / float(NUM_DIRS));
        vec2 dir = vec2(cos(theta), sin(theta));
        float dirMaxOcc = 0.0;

        for (int s = 0; s < NUM_STEPS; ++s) {
            float t = (float(s) + 0.35 + jitterFrac * 0.55) / float(NUM_STEPS);
            vec2 sampleUV = TexCoords + dir * (t * spanUV);
            if (sampleUV.x < 0.001 || sampleUV.x > 0.999 || sampleUV.y < 0.001 || sampleUV.y > 0.999) {
                continue;
            }
            float dS = texture(u_depthTex, sampleUV).r;
            if (dS >= 0.9999) {
                continue;
            }
            vec3 S = reconstructViewPos(sampleUV, dS);
            vec3 delta = S - P;
            float distSq = dot(delta, delta);
            if (distSq > 1e-4 && distSq < rSq) {
                float invDist = inversesqrt(distSq);
                float cosHorizon = dot(N, delta * invDist);
                float horizon = max(cosHorizon - u_bias, 0.0);
                float atten = 1.0 - (distSq / rSq);
                dirMaxOcc = max(dirMaxOcc, horizon * atten * atten);
            }
        }
        occSum += dirMaxOcc;
    }

    float ao = clamp(1.0 - (occSum / float(NUM_DIRS)) * 1.45, 0.0, 1.0);
    FragColor = vec4(ao, ao, ao, 1.0);
}
