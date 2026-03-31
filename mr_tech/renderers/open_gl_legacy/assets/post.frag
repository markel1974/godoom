#version 330 core
out vec4 FragColor;
in vec3 TexCoords;

uniform sampler2D u_hdrBuffer;
uniform sampler2D u_bloomBlur;
uniform float u_exposure;
uniform float u_contrast;
uniform float u_saturation;
uniform float u_bloomIntensity;

// Curva ACES Filmic
vec3 ACESFilm(vec3 x) {
    return clamp((x * (2.51 * x + 0.03)) / (x * (2.43 * x + 0.59) + 0.14), 0.0, 1.0);
}

void main() {
    vec3 color = texture(u_hdrBuffer, TexCoords).rgb;
    vec3 bloom = texture(u_bloomBlur, TexCoords).rgb;

    //const float bloomIntensity = 0.05;
    color += (bloom * u_bloomIntensity);

    // Additive Blend in spazio lineare puro
    color += bloom;

    // 1. Exposure
    color *= u_exposure;

    // 2. Tonemapping ACES (Luminance Only per preservare la cromaticità della texture)
    // Invece di far bruciare i canali RGB al bianco, tonemappiamo solo la luminosità totale.
    float lumaIn = dot(color, vec3(0.2126, 0.7152, 0.0722));
    float lumaOut = ACESFilm(vec3(lumaIn)).r;

    // Riapplica la luminosità tonemappata mantenendo il rapporto esatto dei colori originali
    color = color * (lumaOut / max(lumaIn, 0.0001));

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