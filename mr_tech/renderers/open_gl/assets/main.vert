#version 330 core

invariant gl_Position;

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aNormal;
layout (location = 3) in vec4 aTangent;
layout (location = 4) in vec3 aLightmap;
layout (location = 5) in vec4 aMaterial;    // packed, specBoost, shininess, detailScale
layout (location = 6) in vec4 aExtra;       // detailFactor, envStrength, renderMode, envTintPacked
layout (location = 7) in vec3 aOrigin;      // streaming dynamic only
layout (location = 8) in vec3 aPosNext;     // streaming dynamic only
layout (location = 9) in vec2 aAnimData;    // lerp, yaw (streaming dynamic only)
layout (location = 10) in vec4 aInstanceM0; // instanced model matrix col 0
layout (location = 11) in vec4 aInstanceM1; // instanced model matrix col 1
layout (location = 12) in vec4 aInstanceM2; // instanced model matrix col 2
layout (location = 13) in vec4 aInstanceM3; // instanced model matrix col 3

out vec3 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec3 WorldPos;
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;
out float IsFullbright;
flat out float Material;
out float SpecBoost;
out float Shininess;
out vec2 DetailParams;
out vec3 ViewNormal;
out vec4 ViewTangent;
out vec3 LightmapUV;
out vec4 EnvParams;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;
uniform float u_time;
uniform int u_instanceMode;

vec3 safeNorm3(vec3 v, vec3 fallback) {
    float l2 = dot(v, v);
    return (l2 > 1e-8) ? (v * inversesqrt(l2)) : fallback;
}

vec3 rotateYaw(vec3 v, float cosY, float sinY) {
    float ox = v.x;
    float oy = -v.z;
    float rx = (ox * cosY) - (oy * sinY);
    float ry = (ox * sinY) + (oy * cosY);
    return vec3(rx, v.y, -ry);
}

void renderUI() {
    // HUD / SCREEN SPACE
    float aspect = 1.777;
    float scale = 0.005;
    float ndcX = (aPos.x * scale) / aspect;
    float ndcY = (aPos.y - 8.0) * scale;

    ViewPos = vec3(0.0, 0.0, -1.0);
    WorldPos = aPos;
    FragDepth = 0.0;
    FragPosLightRoom = vec4(0.0);
    FragPosLightFlash = vec4(0.0);
    IsFullbright = 1.0;
    ViewNormal = vec3(0.0, 0.0, 1.0);
    ViewTangent = vec4(1.0, 0.0, 0.0, -1.0);
    gl_Position = vec4(ndcX, ndcY, -0.9, 1.0);
}

void renderBillboard() {
    vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;
    vec3 toCamera = camPos - aOrigin;
    vec3 right, up, forward;
    if (aExtra.z > 1.05) {
        if (length(toCamera) < 0.001) toCamera = vec3(0.0, 0.0, 1.0);
        forward = normalize(toCamera);
        vec3 worldUp = vec3(0.0, 1.0, 0.0);
        if (abs(forward.y) > 0.999) right = vec3(1.0, 0.0, 0.0);
        else right = normalize(cross(worldUp, forward));
        up = cross(forward, right);
    } else {
        toCamera.y = 0.0;
        if (length(toCamera) < 0.001) toCamera = vec3(0.0, 0.0, 1.0);
        forward = normalize(toCamera);
        right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
        up = vec3(0.0, 1.0, 0.0);
    }
    vec4 worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    vec4 viewPos = u_view * worldPos;
    ViewPos = viewPos.xyz;
    WorldPos = worldPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    ViewNormal = mat3(u_view) * forward;
    ViewTangent = vec4(mat3(u_view) * right, -1.0);
    gl_Position = u_projection * viewPos;
}

void renderModel3D() {
    vec3 lPos = mix(aPos, aPosNext, aAnimData.x);
    float cosY = cos(aAnimData.y);
    float sinY = sin(aAnimData.y);
    vec3 rotatedPos = rotateYaw(lPos, cosY, sinY);
    vec3 worldN = rotateYaw(safeNorm3(aNormal, vec3(0.0, 1.0, 0.0)), cosY, sinY);
    vec3 worldT = rotateYaw(aTangent.xyz, cosY, sinY);

    vec4 worldPos = vec4(aOrigin + rotatedPos, 1.0);
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    WorldPos = worldPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    ViewNormal = mat3(u_view) * worldN;
    ViewTangent = vec4(mat3(u_view) * worldT, aTangent.w);
    gl_Position = u_projection * viewPos;
}

void renderLiquid() {
    uint packedMaterial = floatBitsToUint(aMaterial.x);
    uint amplitude = (packedMaterial >> 16u) & 0x7Fu;
    float amplitudeF = float(amplitude);

    vec4 worldPos = vec4(aPos, 1.0);
    worldPos.y += sin(worldPos.x * 0.05 + u_time * 2.0) * amplitudeF;
    worldPos.y += cos(worldPos.z * 0.05 + u_time * 1.5) * amplitudeF;
    TexCoords.x += u_time * 0.1;
    TexCoords.y += u_time * 0.05;
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    WorldPos = worldPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 1.0;
    ViewNormal = mat3(u_view) * safeNorm3(aNormal, vec3(0.0, 1.0, 0.0));
    ViewTangent = vec4(mat3(u_view) * aTangent.xyz, aTangent.w);
    gl_Position = u_projection * viewPos;
}

void renderInstanced() {
    mat4 instMat = mat4(aInstanceM0, aInstanceM1, aInstanceM2, aInstanceM3);
    mat3 instRot = mat3(instMat);
    vec4 worldPos = instMat * vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    WorldPos = worldPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    ViewNormal = mat3(u_view) * (instRot * safeNorm3(aNormal, vec3(0.0, 1.0, 0.0)));
    ViewTangent = vec4(mat3(u_view) * (instRot * aTangent.xyz), aTangent.w);
    gl_Position = u_projection * viewPos;
}

void renderDefault() {
    vec4 worldPos = vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;
    ViewPos = viewPos.xyz;
    WorldPos = worldPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    ViewNormal = mat3(u_view) * safeNorm3(aNormal, vec3(0.0, 1.0, 0.0));
    ViewTangent = vec4(mat3(u_view) * aTangent.xyz, aTangent.w);
    gl_Position = u_projection * viewPos;
}

void main() {
    TexCoords = aTexCoords;
    Material = aMaterial.x;
    SpecBoost = aMaterial.y;
    Shininess = aMaterial.z;
    DetailParams = vec2(aMaterial.w, aExtra.x);
    LightmapUV = aLightmap;

    uint tintU = uint(max(aExtra.w, 0.0) + 0.5);
    vec3 envTint = vec3(
        float(tintU & 0xFFu),
        float((tintU >> 8u) & 0xFFu),
        float((tintU >> 16u) & 0xFFu)
    ) / 255.0;
    EnvParams = vec4(aExtra.y, envTint);

    float renderMode = aExtra.z;
    if (u_instanceMode == 1) {
        renderInstanced();
    } else if (u_instanceMode == 0) {
        if (renderMode > 0.4 && renderMode < 1.0) {
            renderLiquid();
        } else {
            renderDefault();
        }
    } else {
        if (renderMode > 0.4 && renderMode < 1.0) {
            renderLiquid();
        } else if (renderMode >= 1.0 && renderMode < 1.5) {
            renderBillboard();
        } else if (renderMode >= 1.5 && renderMode < 2.5) {
            renderModel3D();
        } else if (renderMode >= 2.5 && renderMode < 3.5) {
            renderUI();
        } else {
            renderDefault();
        }
    }
}
