#version 330 core
in vec3 v_ray;
out vec4 FragColor;

uniform sampler2DArray u_sky[4];
uniform float u_skyLayer;

vec4 getSky(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_sky[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_sky[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_sky[2], vec3(tc.xy, l));
    return texture(u_sky[3], vec3(tc.xy, l));
}

void main() {
    vec3 d = normalize(v_ray);
    const float PI = 3.14159265359;
    float u = atan(d.z, d.x) / (2.0 * PI) + 0.5;
    float v = asin(d.y) / PI + 0.5;

    // Assembliamo U, V e Layer
    FragColor = getSky(vec3(u, v, u_skyLayer));
}