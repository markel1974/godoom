#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoords;
layout (location = 2) in float aLightDist;
layout (location = 3) in vec3 aLightCenterWorld; // Rinominato per coerenza
layout (location = 4) in vec3 aNormal;

out vec2 TexCoords;
out float LightDist;
out float FragDepth;
out vec3 ViewPos;
out vec3 LightCenterView;
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
    LightDist = aLightDist;

    // Trasformazione rigorosa in View Space
    LightCenterView = (u_view * vec4(aLightCenterWorld, 1.0)).xyz;

    vec4 worldPos = vec4(aPos, 1.0);
    vec4 viewPos = u_view * worldPos;
    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);
    NormalView = mat3(u_view) * aNormal;

    vec4 shadowWorldPos = worldPos + vec4(aNormal * 2.0, 0.0);

    FragPosLightRoom = u_roomSpaceMatrix * shadowWorldPos;
    //FragPosLightFlash = u_flashSpaceMatrix * worldPos;
    FragPosLightFlash = u_flashSpaceMatrix * shadowWorldPos;

    gl_Position = u_projection * viewPos;
}