#version 330 core
layout (location = 0) in vec3 aPos;
uniform mat4 u_view;
uniform mat4 u_projection;
uniform int u_debug;

out vec2 v_uv;

void main() {
    int id = gl_VertexID % 6;
    vec2 uv = vec2(0.0);
    if (id == 1) {
        uv = vec2(1.0, 0.0);
    } else if (id == 2 || id == 3) {
        uv = vec2(1.0, 1.0);
    } else if (id == 4) {
        uv = vec2(0.0, 1.0);
    }
    v_uv = uv;

    gl_Position = u_projection * u_view * vec4(aPos, 1.0);
}
