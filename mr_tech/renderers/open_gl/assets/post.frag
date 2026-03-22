#version 330 core
out vec4 FragColor;
in vec2 TexCoords;

uniform sampler2D u_hdrBuffer;
uniform float u_exposure;
uniform float u_contrast;
uniform float u_saturation;

// Curva ACES Filmic
vec3 ACESFilm(vec3 x) {
    return clamp((x * (2.51 * x + 0.03)) / (x * (2.43 * x + 0.59) + 0.14), 0.0, 1.0);
}

void main() {
    vec3 color = texture(u_hdrBuffer, TexCoords).rgb;

    // 1. Exposure
    color *= u_exposure;

    // 2. Tonemapping ACES
    color = ACESFilm(color);

    // 3. Contrast & Saturation
    color = mix(vec3(0.5), color, u_contrast);
    float luma = dot(color, vec3(0.2126, 0.7152, 0.0722));
    color = mix(vec3(luma), color, u_saturation);

    // 4. Gamma Correction (sRGB)
    color = pow(max(color, 0.0), vec3(1.0 / 2.2));

    // Dithering minimo per il banding a 8-bit
    float dither = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) / 255.0;

    FragColor = vec4(color + dither - (0.5 / 255.0), 1.0);
}