#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;
layout (location = 3) in float aIsBillboard;
layout (location = 4) in vec3 aPosNext;
layout (location = 5) in float aLerp;
layout (location = 6) in float aYaw;

out vec3 TexCoords;

uniform mat4 u_lightSpaceMatrix;
uniform mat4 u_view; // Necessario per far orientare l'ombra in base a dove guarda il player

void main()
{
    TexCoords = aTexCoords;
    vec4 worldPos;

    if (aIsBillboard >= 1.0 && aIsBillboard < 1.5) {
        // --- VIEWPOINT BILLBOARDING ---
        vec3 camPos = -transpose(mat3(u_view)) * u_view[3].xyz;
        vec3 toCamera = camPos - aOrigin;
        vec3 right, up;

        if (aIsBillboard > 1.05) {
            // --- BILLBOARD SFERICO ---
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
            // --- BILLBOARD CILINDRICO ---
            toCamera.y = 0.0;
            if (length(toCamera) < 0.001) {
                toCamera = vec3(0.0, 0.0, 1.0);
            }
            vec3 forward = normalize(toCamera);
            right = normalize(cross(vec3(0.0, 1.0, 0.0), forward));
            up = vec3(0.0, 1.0, 0.0);
        }
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aIsBillboard > 1.5) {
        // --- MODELLI 3D (MD2 / MDL) ---
        vec3 lPos = mix(aPos, aPosNext, aLerp);
        float cosY = cos(aYaw);
        float sinY = sin(aYaw);
        float origX = lPos.x;
        float origY = -lPos.z;
        float rotX = (origX * cosY) - (origY * sinY);
        float rotY = (origX * sinY) + (origY * cosY);
        vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
        worldPos = vec4(aOrigin + rotatedPos, 1.0);
    } else {
        // --- GEOMETRIA BSP ---
        worldPos = vec4(aPos, 1.0);
    }

    gl_Position = u_lightSpaceMatrix * worldPos;
}