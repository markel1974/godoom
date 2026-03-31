#version 330 core
layout (location = 0) in vec3 aPos;
out vec3 v_ray;

uniform mat4 u_projection;
uniform mat4 u_view;

void main() {
    gl_Position = vec4(aPos, 1.0, 1.0);

    mat4 invProj = inverse(u_projection);
    mat4 invView = inverse(mat4(mat3(u_view))); // Isola solo la rotazione

    vec4 target = invProj * vec4(aPos.x, aPos.y, 1.0, 1.0);
    v_ray = (invView * vec4(target.xyz, 0.0)).xyz;
}