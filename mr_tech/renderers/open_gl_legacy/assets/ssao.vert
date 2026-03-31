#version 330 core

layout (location = 0) in vec3 aPos;

out vec3 TexCoords;

void main()
{
    // Trasforma le coordinate da [-1, 1] a [0, 1] per il campionamento delle texture
    TexCoords = aPos * 0.5 + 0.5;

    // Posiziona il quad direttamente nello spazio clip
    gl_Position = vec4(aPos, 0.0, 1.0);
}