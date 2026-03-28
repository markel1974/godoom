#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoords;
layout (location = 2) in vec3 aNormal;

out vec2 TexCoords;
out float FragDepth;
out vec3 ViewPos;
out vec3 NormalView;

out vec4 FragPosLightRoom;
out vec4 FragPosLightFlash;

uniform mat4 u_view;
uniform mat4 u_projection;
uniform mat4 u_roomSpaceMatrix;
uniform mat4 u_flashSpaceMatrix;

void main()
{
    TexCoords = aTexCoords;

    vec4 worldPos = vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;

    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    NormalView = mat3(u_view) * aNormal;

    vec4 shadowWorldPos = worldPos + vec4(aNormal * 2.0, 0.0);

    FragPosLightRoom = u_roomSpaceMatrix * shadowWorldPos;
    FragPosLightFlash = u_flashSpaceMatrix * worldPos;

    gl_Position = u_projection * viewPos;
}