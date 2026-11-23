#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 6) in vec4 aExtra;       // z = renderMode
layout (location = 7) in vec3 aOrigin;      // streaming dynamic only
layout (location = 8) in vec3 aPosNext;     // streaming dynamic only
layout (location = 9) in vec2 aAnimData;    // x = lerp, y = yaw
layout (location = 10) in vec4 aInstanceM0;
layout (location = 11) in vec4 aInstanceM1;
layout (location = 12) in vec4 aInstanceM2;
layout (location = 13) in vec4 aInstanceM3;

out vec3 TexCoords;

uniform mat4 u_lightSpaceMatrix;
uniform mat4 u_view; // Necessario per far orientare l'ombra in base a dove guarda il player
uniform float u_time;
uniform int u_instanceMode;

void render3DModels() {
    //MODELLI 3D
    vec3 lPos = mix(aPos, aPosNext, aAnimData.x);
    float cosY = cos(aAnimData.y);
    float sinY = sin(aAnimData.y);
    float origX = lPos.x;
    float origY = -lPos.z;
    float rotX = (origX * cosY) - (origY * sinY);
    float rotY = (origX * sinY) + (origY * cosY);
    vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
    vec4 worldPos = vec4(aOrigin + rotatedPos, 1.0);

    gl_Position = u_lightSpaceMatrix * worldPos;
}

void renderBillboarding() {
    // VIEWPOINT BILLBOARDING
    vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;
    vec3 toCamera = camPos - aOrigin;
    vec3 right, up;
    if (aExtra.z > 1.05) {
        // BILLBOARD SFERICO
        if (length(toCamera) < 0.001) {
            toCamera = vec3(0.0, 0.0, 1.0);
        }
        vec3 forward = normalize(toCamera);
        vec3 worldUp = vec3(0.0, 1.0, 0.0);
        if (abs(forward.y) > 0.999) {
            right = vec3(1.0, 0.0, 0.0);
        } else {
            right = normalize(cross(worldUp, forward));
        }
        up = cross(forward, right);
    } else {
        // BILLBOARD CILINDRICO
        toCamera.y = 0.0;
        if (length(toCamera) < 0.001) {
            toCamera = vec3(0.0, 0.0, 1.0);
        }
        vec3 forward = normalize(toCamera);
        right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
        up = vec3(0.0, 1.0, 0.0);
    }
    vec4 worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    gl_Position = u_lightSpaceMatrix * worldPos;
}

void renderLiquid() {
    // MATERIALI ANIMATI O TRASPARENTI
    vec4 worldPos = vec4(aPos, 1.0);
    worldPos.y += sin(worldPos.x * 0.05 + u_time * 2.0) * 2.0;
    worldPos.y += cos(worldPos.z * 0.05 + u_time * 1.5) * 2.0;
    gl_Position = u_lightSpaceMatrix * worldPos;
}

void renderInstanced() {
    mat4 instMat = mat4(aInstanceM0, aInstanceM1, aInstanceM2, aInstanceM3);
    vec4 worldPos = instMat * vec4(aPos, 1.0);
    gl_Position = u_lightSpaceMatrix * worldPos;
}

void renderAll() {
    // --- GEOMETRIA BSP / STATIC ---
    vec4 worldPos = vec4(aPos, 1.0);
    gl_Position = u_lightSpaceMatrix * worldPos;
}

void main() {
    TexCoords = aTexCoords;
    float renderMode = aExtra.z;
    if (u_instanceMode == 1) {
        renderInstanced();
    } else if (u_instanceMode == 0) {
        if (renderMode > 0.4 && renderMode < 0.6) {
            renderLiquid();
        } else {
            renderAll();
        }
    } else {
        if (renderMode >= 1.0 && renderMode < 1.5) {
            renderBillboarding();
        } else if (renderMode >= 1.5) {
            render3DModels();
        } else if (renderMode > 0.4 && renderMode < 0.6) {
            renderLiquid();
        } else {
            renderAll();
        }
    }
}