package open_gl

const shaderVertex = `
#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoords;
layout (location = 2) in float aLightDist;

out vec2 TexCoords;
out float LightDist;

uniform mat4 u_view;
uniform mat4 u_projection;

void main()
{
    TexCoords = aTexCoords;
    LightDist = aLightDist;
    gl_Position = u_projection * u_view * vec4(aPos, 1.0);
}
`
