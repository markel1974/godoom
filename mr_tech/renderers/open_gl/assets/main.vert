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
    if (aIsBillboard > 0.5) {
        // Estrazione dei vettori Right e Up dalla View Matrix
        // In una matrice View, le righe (o le colonne della trasposta)
        // rappresentano gli assi della camera nello spazio del mondo.
        vec3 right = vec3(u_view[0][0], u_view[1][0], u_view[2][0]);
        vec3 up    = vec3(u_view[0][1], u_view[1][1], u_view[2][1]);

        // Calcolo della posizione assoluta: Origin + (Right * x_local) + (Up * y_local)
        // aPos.x e aPos.y contengono gli scostamenti definiti in GetVertices()
        worldPos = vec4(aOrigin + (right * aPos.x) + (up * aPos.y), 1.0);
    } else {
        // Geometria statica o mesh dinamica (MD2) già trasformata
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