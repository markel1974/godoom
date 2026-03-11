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

    // --- 1. SPOTLIGHT ---
    vec3 lightVector = LightCenterView - ViewPos;
    vec3 L = normalize(lightVector);
    vec3 spotDir = vec3(0.0, -1.0, 0.0);
    float cosTheta = dot(-L, spotDir);

    // 0.50 = cos(60°), apertura totale della luce piena a 120 gradi
    // 0.30 = cos(72.5°), penombra esterna che sfuma fino a 145 gradi totali
    float spotIntensity = smoothstep(0.30, 0.50, cosTheta);
    //float spotIntensity = smoothstep(0.70, 0.85, cosTheta);

    // --- 2. NORMALE BASE E BUMP ---
    vec3 N = NormalView;
    if (dot(N, N) < 0.01) {
        N = vec3(0.0, 1.0, 0.0);
    } else {
        N = normalize(N);
    }

    vec3 finalNormal = N;
    float bumpFactor = 1.0;
    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;

    // Applica TBN solo se c'è una normal map
    if (length(mapColor) > 0.1) {
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

        finalNormal = normalize(TBN * mapNormal);
    }

    // Il bumpFactor influisce sulla luce diffusa, indipendentemente dalla normal map
    bumpFactor = (max(dot(finalNormal, L), 0.0) * 0.2) + 1.0;

    // --- 3. RIFLESSO SPECULARE (BLINN-PHONG BILANCIATO) ---
    vec3 V = normalize(-ViewPos);
    vec3 H = normalize(L + V);

    float NdotH = max(dot(finalNormal, H), 0.0);

    // Identifica piani orizzontali in View Space
    float isHorizontal = step(0.8, abs(finalNormal.y));

    // Muri: shininess 16.0 (riflesso più concentrato)
    // Pavimento: shininess 4.0 (lobo allargato per intercettare la camera radente)
    float shininess = mix(1.0, 4.0, isHorizontal);

    // Moltiplicatore contenuto, senza luma della texture base
    float specBoost = mix(0.5, 1.5, isHorizontal);

    // Il clamp impedisce categoricamente l'over-saturazione (valori > 1.0)
    float specular = clamp(pow(NdotH, shininess) * specBoost, 0.0, 1.0);

    // --- 4. DECADIMENTO ---
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float visibilityMultiplier = 0.1;
    float falloff = exp(-FragDepth * decayRate * visibilityMultiplier);

    // --- 5. COMPOSIZIONE ---
    float lightMix = 0.3 + (spotIntensity * 1.7);

    // Highlight speculare modulato ESCLUSIVAMENTE dal cono di luce
    vec3 specularHighlight = vec3(specular * spotIntensity);

    vec3 baseColor = texColor.rgb * bumpFactor;

    // Aggiunta lineare del riflesso post-calcolo diffusivo, scalata per il decadimento
    vec3 litColor = (baseColor * lightMix + specularHighlight) * falloff;

    vec3 finalColor = pow(max(litColor, 0.0), vec3(0.8));

    FragColor = vec4(finalColor, texColor.a);
}