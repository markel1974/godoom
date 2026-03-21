#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoords;
layout (location = 2) in float aLightDist;
layout (location = 3) in vec3 aLightCenterView;
layout (location = 4) in vec3 aNormal;

out vec2 TexCoords;
out float LightDist;
out float FragDepth;
out vec3 ViewPos;
out vec3 LightCenterView;
out vec3 NormalView;

// Output verso il Fragment Shader per il campionamento ombra
out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

void main()
{
    TexCoords = aTexCoords;
    LightDist = aLightDist;
    LightCenterView = aLightCenterView;

    vec4 worldPos = vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    NormalView = mat3(u_view) * aNormal;

    // Trasformazione del vertice nello spazio ottico delle due luci
    FragPosLightRoom = u_roomSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;

    gl_Position = u_projection * viewPos;
}