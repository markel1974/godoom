#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;
in vec3 ViewPos;
in vec3 LightCenterView;
in vec3 NormalView;

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform float u_ambient_light;

void main()
{
    vec4 texColor = texture(u_texture, TexCoords, -0.5);

    if(texColor.a < 0.1) {
        discard;
    }

    float bumpFactor = 1.0;
    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;

    if (length(mapColor) > 0.1) {
        vec3 N = NormalView;
        if (dot(N, N) < 0.01) {
            N = vec3(0.0, 1.0, 0.0);
        } else {
            N = normalize(N);
        }

        vec3 mapNormal = mapColor * 2.0 - 1.0;

        vec3 dp1 = dFdx(ViewPos);
        vec3 dp2 = dFdy(ViewPos);
        vec2 duv1 = dFdx(TexCoords);
        vec2 duv2 = dFdy(TexCoords);

        vec3 dp2perp = cross(dp2, N);
        vec3 dp1perp = cross(N, dp1);
        vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
        vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

        float det = max(dot(T,T), dot(B,B));
        float invmax = inversesqrt(det + 0.0001);
        mat3 TBN = mat3(T * invmax, B * invmax, N);

        vec3 finalNormal = normalize(TBN * mapNormal);

        // Calcolo vettoriale puro basato ESATTAMENTE sulle coordinate del tuo modello
        vec3 lightDir = vec3(0.0, 1.0, 0.0);
        vec3 lightVector = LightCenterView - ViewPos;
        if (dot(lightVector, lightVector) > 0.01) {
            lightDir = normalize(lightVector);
        }

        // Mappatura [-1.0, 1.0] -> [0.8, 1.2] per modulare la texture base
        bumpFactor = (dot(finalNormal, lightDir) * 0.2) + 1.0;
    }

    // La tua equazione di decadimento intatta
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float visibilityMultiplier = 0.1;
    float falloff = exp(-FragDepth * decayRate * visibilityMultiplier);

    // Applicazione del bump e del decadimento
    vec3 litColor = (texColor.rgb * bumpFactor) * falloff;
    vec3 finalColor = pow(max(litColor, 0.0), vec3(0.8));

    FragColor = vec4(finalColor, texColor.a);
}