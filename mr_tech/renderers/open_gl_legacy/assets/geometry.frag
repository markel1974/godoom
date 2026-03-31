#version 330 core
layout (location = 0) out vec4 gPositionDepth;
layout (location = 1) out vec4 gNormal;

in vec3 ViewPos;
in vec3 NormalView;
in vec3 TexCoords;
in float FragDepth;

uniform sampler2D u_texture;

void main() {
    // Rispetta l'alpha discard per i portali/sprite
    if(texture(u_texture, TexCoords).a < 0.5) discard;

    // Attachment 0
    gPositionDepth = vec4(ViewPos, FragDepth);
    // Attachment 1
    gNormal = vec4(normalize(NormalView), 1.0);
}