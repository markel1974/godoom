#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;
layout (location = 3) in float aRenderMode;
layout (location = 4) in vec3 aPosNext;
layout (location = 5) in float aLerp;
layout (location = 6) in float aYaw;

out vec3 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;
out float IsFullbright;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;
uniform float u_time;

void renderInterface() {
    // HUD / SCREEN SPACE (2D PURO)
    // Bypassiamo totalmente matrici e ritorni anticipati
    float aspect = 1.777;
    float scale = 0.005;
    float ndcX = (aPos.x * scale) / aspect;
    float ndcY = (aPos.y - 8.0) * scale;

    ViewPos = vec3(0.0, 0.0, -1.0);
    FragDepth = 0.0;
    FragPosLightRoom = vec4(0.0);
    FragPosLightFlash = vec4(0.0);
    IsFullbright = 1.0;
    gl_Position = vec4(ndcX, ndcY, -0.9, 1.0);
}

void renderBillboard() {
    vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;
    vec3 toCamera = camPos - aOrigin;
    vec3 right, up;
    if (aRenderMode > 1.05) {
        if (length(toCamera) < 0.001) toCamera = vec3(0.0, 0.0, 1.0);
        vec3 forward = normalize(toCamera);
        vec3 worldUp = vec3(0.0, 1.0, 0.0);
        if (abs(forward.y) > 0.999) right = vec3(1.0, 0.0, 0.0);
        else right = normalize(cross(worldUp, forward));
        up = cross(forward, right);
    } else {
        toCamera.y = 0.0;
        if (length(toCamera) < 0.001) toCamera = vec3(0.0, 0.0, 1.0);
        vec3 forward = normalize(toCamera);
        right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
        up = vec3(0.0, 1.0, 0.0);
    }
    vec4 worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    vec4 viewPos = u_view * worldPos;
    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    gl_Position = u_projection * viewPos;
}

void renderModel3D() {
    vec3 lPos = mix(aPos, aPosNext, aLerp);
    float cosY = cos(aYaw);
    float sinY = sin(aYaw);
    float origX = lPos.x;
    float origY = -lPos.z;
    float rotX = (origX * cosY) - (origY * sinY);
    float rotY = (origX * sinY) + (origY * cosY);
    vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
    vec4 worldPos = vec4(aOrigin + rotatedPos, 1.0);
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    gl_Position = u_projection * viewPos;
}

void renderAnimated() {
    vec4 worldPos = vec4(aPos, 1.0);
    worldPos.y += sin(worldPos.x * 0.05 + u_time * 2.0) * 2.0;
    worldPos.y += cos(worldPos.z * 0.05 + u_time * 1.5) * 2.0;
    TexCoords.x += u_time * 0.1;
    TexCoords.y += u_time * 0.05;
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 1.0;
    gl_Position = u_projection * viewPos;
}

void renderDefault() {
    vec4 worldPos = vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;
    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    IsFullbright = 0.0;
    gl_Position = u_projection * viewPos;
}

void main() {
    TexCoords = aTexCoords;
    if (aRenderMode >= 2.5 && aRenderMode < 3.5) {
        renderInterface();
        return;
    }
    if (aRenderMode >= 1.0 && aRenderMode < 1.5) {
        renderBillboard();
    } else if (aRenderMode > 1.5 && aRenderMode < 2.5) {
        renderModel3D();
    } else if (aRenderMode > 0.4 && aRenderMode < 0.6) {
        renderAnimated();
    } else {
        renderDefault();
    }
}
