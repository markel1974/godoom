#version 330 core
out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D u_srcTex;
uniform vec2 u_texelSize;
uniform int u_mode;            // 0 = Jimenez 13-tap downsample, 1 = 9-tap tent upsample
uniform int u_karis;           // 1 = apply Karis average on mip 0 to prevent fireflies
uniform float u_filterRadius;
uniform float u_upsampleWeight;

float karisWeight(vec3 c) {
    float luma = dot(max(c, vec3(0.0)), vec3(0.2126, 0.7152, 0.0722));
    return 1.0 / (1.0 + luma);
}

vec3 downsampleBloom() {
    float x = u_texelSize.x;
    float y = u_texelSize.y;

    vec3 a = texture(u_srcTex, vec2(TexCoords.x - 2.0 * x, TexCoords.y + 2.0 * y)).rgb;
    vec3 b = texture(u_srcTex, vec2(TexCoords.x,           TexCoords.y + 2.0 * y)).rgb;
    vec3 c = texture(u_srcTex, vec2(TexCoords.x + 2.0 * x, TexCoords.y + 2.0 * y)).rgb;

    vec3 d = texture(u_srcTex, vec2(TexCoords.x - 2.0 * x, TexCoords.y)).rgb;
    vec3 e = texture(u_srcTex, vec2(TexCoords.x,           TexCoords.y)).rgb;
    vec3 f = texture(u_srcTex, vec2(TexCoords.x + 2.0 * x, TexCoords.y)).rgb;

    vec3 g = texture(u_srcTex, vec2(TexCoords.x - 2.0 * x, TexCoords.y - 2.0 * y)).rgb;
    vec3 h = texture(u_srcTex, vec2(TexCoords.x,           TexCoords.y - 2.0 * y)).rgb;
    vec3 i = texture(u_srcTex, vec2(TexCoords.x + 2.0 * x, TexCoords.y - 2.0 * y)).rgb;

    vec3 j = texture(u_srcTex, vec2(TexCoords.x - x, TexCoords.y + y)).rgb;
    vec3 k = texture(u_srcTex, vec2(TexCoords.x + x, TexCoords.y + y)).rgb;
    vec3 l = texture(u_srcTex, vec2(TexCoords.x - x, TexCoords.y - y)).rgb;
    vec3 m = texture(u_srcTex, vec2(TexCoords.x + x, TexCoords.y - y)).rgb;

    if (u_karis == 1) {
        vec3 g0 = (j + k + l + m) * 0.25;
        vec3 g1 = (a + b + d + e) * 0.25;
        vec3 g2 = (b + c + e + f) * 0.25;
        vec3 g3 = (d + e + g + h) * 0.25;
        vec3 g4 = (e + f + h + i) * 0.25;

        float kw0 = karisWeight(g0) * 0.50;
        float kw1 = karisWeight(g1) * 0.125;
        float kw2 = karisWeight(g2) * 0.125;
        float kw3 = karisWeight(g3) * 0.125;
        float kw4 = karisWeight(g4) * 0.125;

        float wSum = max(kw0 + kw1 + kw2 + kw3 + kw4, 1e-4);
        vec3 res = (g0 * kw0 + g1 * kw1 + g2 * kw2 + g3 * kw3 + g4 * kw4) / wSum;
        return max(res, vec3(0.0));
    }

    vec3 res = e * 0.125;
    res += (a + c + g + i) * 0.03125;
    res += (b + d + f + h) * 0.0625;
    res += (j + k + l + m) * 0.125;
    return max(res, vec3(0.0));
}

vec3 upsampleBloom() {
    float x = u_texelSize.x * u_filterRadius;
    float y = u_texelSize.y * u_filterRadius;

    vec3 a = texture(u_srcTex, vec2(TexCoords.x - x, TexCoords.y + y)).rgb;
    vec3 b = texture(u_srcTex, vec2(TexCoords.x,     TexCoords.y + y)).rgb;
    vec3 c = texture(u_srcTex, vec2(TexCoords.x + x, TexCoords.y + y)).rgb;

    vec3 d = texture(u_srcTex, vec2(TexCoords.x - x, TexCoords.y)).rgb;
    vec3 e = texture(u_srcTex, vec2(TexCoords.x,     TexCoords.y)).rgb;
    vec3 f = texture(u_srcTex, vec2(TexCoords.x + x, TexCoords.y)).rgb;

    vec3 g = texture(u_srcTex, vec2(TexCoords.x - x, TexCoords.y - y)).rgb;
    vec3 h = texture(u_srcTex, vec2(TexCoords.x,     TexCoords.y - y)).rgb;
    vec3 i = texture(u_srcTex, vec2(TexCoords.x + x, TexCoords.y - y)).rgb;

    vec3 tent = e * 4.0 + (b + d + f + h) * 2.0 + (a + c + g + i);
    return max(tent * (1.0 / 16.0) * u_upsampleWeight, vec3(0.0));
}

void main() {
    vec3 color = (u_mode == 0) ? downsampleBloom() : upsampleBloom();
    FragColor = vec4(color, 1.0);
}
