#version 330 core
in vec3 v_ray;
layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_sky[4];
uniform float u_skyLayer;
uniform float u_scrollU;
uniform float u_scrollV;

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
    float u = atan(d.z, d.x) / (2.0 * PI) + 0.5 + u_scrollU;
    float v = asin(d.y) / PI + 0.5 + u_scrollV;

    // Assembliamo U, V e Layer
    vec4 skyCol = getSky(vec3(u, v, u_skyLayer));
    FragColor = skyCol;
    BrightColor = vec4(dot(skyCol.rgb, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? skyCol.rgb : vec3(0.0), 1.0);
}