#version 330 core
layout (location = 0) in vec2 aPos;
out vec3 v_ray;
uniform mat4 u_inv_proj_view;

void main() {
    // Z a 0.99999 per massimizzare la profondità e superare l'Early-Z cull
    gl_Position = vec4(aPos, 0.99999, 1.0);

    // Unproject dal NDC allo spazio mondo
    vec4 target = u_inv_proj_view * vec4(aPos.x, aPos.y, 1.0, 1.0);
    v_ray = target.xyz / target.w;
}