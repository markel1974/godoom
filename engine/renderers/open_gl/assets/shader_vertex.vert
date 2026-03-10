
#version 330 core

layout (location = 0) in vec3 aPos;
layout (location = 1) in vec2 aTexCoords;
layout (location = 2) in float aLightDist;

out vec2 TexCoords;
out float LightDist;
out float FragDepth;
out vec3 ViewPos;

uniform mat4 u_view;
uniform mat4 u_projection;

void main()
{
    TexCoords = aTexCoords;
    LightDist = aLightDist;

    vec4 viewPos = u_view * vec4(aPos, 1.0);
    ViewPos = viewPos.xyz;
    FragDepth = abs(viewPos.z);

    gl_Position = u_projection * viewPos;
}

