package open_gl

const shaderFragment = `
#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;

uniform sampler2D u_texture;

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);

    // Discard per le middle textures con trasparenze (masking)
    if(texColor.a < 0.1) {
        discard;
    }

    // Attenuazione fotometrica: 0.0 = luce ambientale massima (nessun decadimento)
    // Regola il clamp o l'equazione in base al tuning di vi.LightDistance
    float attenuation = clamp(1.0 - LightDist, 0.0, 1.0);

    FragColor = vec4(texColor.rgb * attenuation, texColor.a);
}
`
