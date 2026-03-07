package open_gl

const shaderFragment = `
#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;

uniform sampler2D u_texture;
uniform float u_ambient_light;

void main()
{
    vec4 texColor = texture(u_texture, TexCoords, -0.5);

    if(texColor.a < 0.1) {
        discard;
    }

    // Acquisizione del rate di decadimento (settore o globale)
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;

    // Fattore di scala globale del tuo mondo (prima era 5.0)
    // Se la scena è troppo scura, riduci questo valore (es. 1.0 o 0.5)
    float visibilityMultiplier = 5.0; 

    // Legge di Beer-Lambert: 1.0 a distanza 0, scende asintoticamente a 0.0 (buio)
    float falloff = exp(-FragDepth * decayRate * visibilityMultiplier);
    
    // Attenuazione
    vec3 litColor = texColor.rgb * falloff;

    // Tonemapping/Gamma (max() previene errori hardware con basi negative in pow)
    vec3 finalColor = pow(max(litColor, 0.0), vec3(0.8));

    FragColor = vec4(finalColor, texColor.a);
}
`
