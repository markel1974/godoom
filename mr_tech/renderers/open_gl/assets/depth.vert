#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aTexCoords;
layout (location = 2) in vec3 aOrigin;        // Aggiunto
layout (location = 3) in float aIsBillboard;  // Aggiunto

out vec3 TexCoords;

uniform mat4 u_lightSpaceMatrix;
// Passiamo la matrice di View del giocatore per estrarre Right e Up
// Questo garantisce che lo sprite proietti l'ombra basandosi sulla sua rotazione verso il player
uniform mat4 u_view;

void main() {
    TexCoords = aTexCoords;
    vec4 worldPos;

    if (aIsBillboard == 1.0) {
        // Sprite 2.5D: Stessa logica del Main Shader
        vec3 right = vec3(u_view[0][0], u_view[1][0], u_view[2][0]);
        vec3 up    = vec3(u_view[0][1], u_view[1][1], u_view[2][1]);
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aIsBillboard > 1.5) {
        // MD2: Sommiamo l'origine assoluta
        worldPos = vec4(aOrigin + aPos, 1.0);
    } else {
        // BSP: Coordinate già assolute
        worldPos = vec4(aPos, 1.0);
    }

    gl_Position = u_lightSpaceMatrix * worldPos;
}