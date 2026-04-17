#version 330 core

layout (location = 0) in vec3 aPos;           // Offset locale (se billboard) o mesh cruda
layout (location = 1) in vec3 aTexCoords;     // U, V, Layer
layout (location = 2) in vec3 aOrigin;        // Posizione mondo dell'entità
layout (location = 3) in float aIsBillboard;  // 1.0 = Sprite, 0.0 = Mesh

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
        // Sprite 2.5D (Billboard classico)
        vec3 right = vec3(u_view[0][0], u_view[1][0], u_view[2][0]);
        vec3 up    = vec3(u_view[0][1], u_view[1][1], u_view[2][1]);
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else if (aIsBillboard > 1.5) {
        // Flag 2.0: Modelli 3D dinamici (MD2/MDL)
        // La GPU somma la posizione locale (ruotata su CPU) all'origine assoluta
        worldPos = vec4(aOrigin + aPos, 1.0);
    } else {
        // Flag 0.0: Geometria BSP della mappa (coordinate già assolute)
        worldPos = vec4(aPos, 1.0);
    }

    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);

    // Manteniamo il calcolo per le ombre e le luci dinamiche
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;

    gl_Position = u_projection * viewPos;
}