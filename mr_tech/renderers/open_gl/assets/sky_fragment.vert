#version 330 core
in vec3 v_ray;
out vec4 FragColor;
uniform sampler2D u_sky;

void main() {
    vec3 d = normalize(v_ray);

    // Mappatura cilindrica (Equirettangolare)
    const float PI = 3.14159265359;
    float u = atan(d.z, d.x) / (2.0 * PI) + 0.5;
    float v = asin(d.y) / PI + 0.5;

    FragColor = texture(u_sky, vec2(u, v));
}