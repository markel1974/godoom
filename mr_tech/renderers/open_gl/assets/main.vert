#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;
layout (location = 3) in float aIsBillboard;
layout (location = 4) in vec3 aPosNext;
layout (location = 5) in float aLerp;
layout (location = 6) in float aYaw;

out vec3 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

void main()
{
    TexCoords = aTexCoords;

    vec4 worldPos;

    if (aIsBillboard == 1.0) {
        // Sprite 2.5D
        vec3 right = vec3(u_view[0][0], u_view[1][0], u_view[2][0]);
        vec3 up    = vec3(u_view[0][1], u_view[1][1], u_view[2][1]);
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aIsBillboard > 1.5) {
        // Modelli 3D (MD2/MDL)
        // 1. Interpolazione Hardware (Costo zero sulla GPU)
        vec3 lPos = mix(aPos, aPosNext, aLerp);
        // 2. Rotazione Yaw (Orizzontale)
        float cosY = cos(aYaw);
        float sinY = sin(aYaw);
        // Recuperiamo gli assi originali del MD2 per la rotazione:
        // Nel builder Go mappiamo: X -> x, Z -> y, -Y -> z
        float origX = lPos.x;
        float origY = -lPos.z; // Invertiamo per ritrovare la Y originale (profondità)
        // Applichiamo la rotazione standard 2D (come faceva la tua CPU)
        float rotX = (origX * cosY) - (origY * sinY);
        float rotY = (origX * sinY) + (origY * cosY);
        // Riapplichiamo il mapping OpenGL (X, Y=altezza immutata, Z=-rotY)
        vec3 rotatedPos = vec3(rotX, lPos.y, -rotY);
        // 3. Somma l'origine assoluta
        worldPos = vec4(aOrigin + rotatedPos, 1.0);
    } else {
        // Geometria BSP statica
        worldPos = vec4(aPos, 1.0);
    }

    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);

    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;

    gl_Position = u_projection * viewPos;
}