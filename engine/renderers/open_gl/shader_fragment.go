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

    // Attenuazione pura basata sulla profondità Z (lineare, senza angoli)
    if (LightDist >= 0.0) {
        depthFactor = FragDepth * visibility * LightDist;
    } else {
        depthFactor = FragDepth * visibility * u_ambient_light;
    }

    float fogFactor = clamp(depthFactor, 0.0, 1.0);

    // Particellare stocastico leggero
    float particleNoise = random(gl_FragCoord.xy + fract(u_time)) * 0.10;
    vec3 fogColor = vec3(particleNoise); 
    
    // Mix lineare tra il colore pieno della texture e il buio "sporco" in lontananza
    vec3 finalColor = mix(texColor.rgb, fogColor, fogFactor);

    FragColor = vec4(finalColor, texColor.a);
}
`
