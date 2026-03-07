package open_gl

const shaderFragment = `
#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;

uniform sampler2D u_texture;
uniform float u_ambient_light;
uniform float u_time;

// Generatore stocastico per la simulazione del pulviscolo in screen-space
float random(vec2 st) {
    return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123);
}

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);

    if(texColor.a < 0.1) {
        discard;
    }

    float visibility = 5.0; 
    float depthFactor = 0.0;

    if (LightDist >= 0.0) {
        depthFactor = FragDepth * visibility * LightDist;
    } else {
        depthFactor = FragDepth * visibility * u_ambient_light;
    }

    // clamping della densità volumetrica
    float fogFactor = clamp(depthFactor, 0.0, 1.0);

    // Generazione particolato: animato nel tempo e scalato in intensità (0.15)
    float particleNoise = random(gl_FragCoord.xy + fract(u_time)) * 0.15;
    
    // In-scattering: colore del volume illuminato sommato al particolato
    vec3 fogColor = vec3(0.02) + vec3(particleNoise);
    
    // Blending non-lineare tra la texture e la diffusione volumetrica
    vec3 finalColor = mix(texColor.rgb, fogColor, fogFactor);

    FragColor = vec4(finalColor, texColor.a);
}
`
