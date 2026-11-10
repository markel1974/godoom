#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 v_uv;
uniform int u_debug;

void main() {
    if (u_debug != 0) {
        // Distanza dai bordi della faccia del cubo (da 0.0 sui bordi a 0.5 al centro)
        vec2 d = min(v_uv, vec2(1.0) - v_uv);
        float edgeDist = min(d.x, d.y);

        if (edgeDist < 0.04) {
            // Bordo neon verde/giallo brillante per delineare il Bounding Box
            FragColor = vec4(0.0, 1.0, 0.2, 0.95);
            BrightColor = vec4(0.0, 1.0, 0.2, 1.0);
        } else {
            // Facce semitrasparenti per vedere attraverso l'oggetto all'interno
            FragColor = vec4(0.0, 0.8, 0.2, 0.10);
            BrightColor = vec4(0.0);
        }
    } else {
        FragColor = vec4(1.0);
        BrightColor = vec4(0.0);
    }
}
