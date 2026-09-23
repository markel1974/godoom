#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;

uniform sampler2DArray u_texture[4];

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_texture[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_texture[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_texture[2], vec3(tc.xy, l));
    return texture(u_texture[3], vec3(tc.xy, l));
}

void main()
{
    vec4 texColor = getDiffuse(TexCoords);

    // Fill-rate optimization: discard completely black/transparent pixels
    if (texColor.a < 0.05 && (texColor.r + texColor.g + texColor.b < 0.05)) {
        discard;
    }
    // Also discard if additive color is completely black, as it won't add anything
    if (texColor.r + texColor.g + texColor.b < 0.05) {
        discard;
    }

    // By NOT converting the sRGB texture to linear space with pow(2.2), we prevent
    // dark values (like the faint gradients in flames and beams) from being crushed to 0.
    // Treating sRGB values as linear directly gives them a massive brightness boost
    // when the final HDR buffer is gamma-corrected, which is exactly what we want
    // for intense additive glowing effects, while preserving their original color ratio.
    vec3 albedo = texColor.rgb * 8.0;

    // Additive materials do not evaluate lights or SSAO
    FragColor = vec4(albedo, texColor.a);

    // Ensure flames bloom slightly to look glowing
    BrightColor = vec4(albedo * 1.5, 1.0);
}
