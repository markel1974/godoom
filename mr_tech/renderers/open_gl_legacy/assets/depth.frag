#version 330 core
in vec3 TexCoords;

uniform sampler2D u_texture;

void main() {
    // Indispensabile per proiettare correttamente le ombre di sprite e grate
    if(texture(u_texture, TexCoords).a < 0.5) {
        discard;
    }
    // gl_FragDepth viene gestito implicitamente dall'hardware
}