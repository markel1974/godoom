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

    //vec3 albedo = pow(texColor.rgb, vec3(2.2)); TEST PASSIAMO a 0.1 per renderlo visibile
    vec3 albedo = pow(texColor.rgb, vec3(0.1));

    // Additive materials do not evaluate lights or SSAO
    FragColor = vec4(albedo, texColor.a);

    // Ensure flames bloom slightly to look glowing
    BrightColor = vec4(albedo * 1.5, 1.0);
}
