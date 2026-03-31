#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;

uniform sampler2DArray u_texture;
uniform sampler2DArray u_normalMap;
uniform sampler2DArray u_emissiveMap;
uniform sampler2D u_ssao;
uniform vec2 u_screenResolution;
uniform float u_emissiveIntensity;
uniform float u_aoFactor;

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    // CAMPIONAMENTO SSAO
    float ao = texture(u_ssao, screenUV).r;

    // FIX: Ammorbidiamo l'occlusione ambientale ed evitiamo che oscuri interi poligoni.
    // L'SSAO ora modula dolcemente la luminosità, senza farla mai scendere a livelli di nero netto (es. bloccato al 60%).
    //float ambientOcclusion = mix(1.0, ao, u_aoFactor);
    //float linearAmbient = max(ambientOcclusion, 0.6);
    float linearAmbient = max(pow(ao * u_aoFactor, 2.2), 0.05);

    vec3 emissive = texture(u_emissiveMap, TexCoords).rgb * u_emissiveIntensity * edgeFade;
    // Colore base pulito senza poligoni anneriti
    vec3 baseColor = (albedo * linearAmbient) + emissive;

    FragColor = vec4(baseColor, texColor.a);
    BrightColor = vec4(dot(baseColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? baseColor : vec3(0.0), 1.0);
}