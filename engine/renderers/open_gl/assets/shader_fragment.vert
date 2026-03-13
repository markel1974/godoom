#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;
in vec3 ViewPos;           // Posizione del frammento (World Space)
in vec3 LightCenterView;    // Centro della luce del settore (World Space)
in vec3 NormalView;         // Normale geometrica (World Space)

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform float u_ambient_light;

// Uniform per la torcia e il calcolo speculare corretto
uniform vec3 u_cameraPos;
uniform vec3 u_cameraFront;

void main()
{
    vec4 texColor = texture(u_texture, TexCoords, -0.5);

    if(texColor.a < 0.1) {
        discard;
    }

    // --- 1. VETTORI DI ILLUMINAZIONE ---

    // A. Luce del Settore (Faretto zenitale dall'alto)
    vec3 lightVectorRoom = LightCenterView - ViewPos;
    vec3 L_room = normalize(lightVectorRoom);
    vec3 spotDirRoom = vec3(0.0, -1.0, 0.0); // Punta verso il basso
    float cosThetaRoom = dot(-L_room, spotDirRoom);
    float roomSpotIntensity = smoothstep(0.30, 0.50, cosThetaRoom);

    // B. Torcia (In View Space la camera è all'origine 0,0,0)
    vec3 L_flash = normalize(-ViewPos);
    // In View Space la camera punta sempre verso -Z
    vec3 viewFront = vec3(0.0, 0.0, -1.0);
    float cosThetaFlash = dot(-L_flash, viewFront); // Equivalente a L_flash.z
    float flashIntensity = smoothstep(0.85, 0.95, cosThetaFlash);

    // --- 2. NORMALE E BUMP MAPPING (TBN) ---
    vec3 N = NormalView;
    if (dot(N, N) < 0.01) {
        N = vec3(0.0, 1.0, 0.0);
    } else {
        N = normalize(N);
    }

    vec3 finalNormal = N;
    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;

    // Generazione TBN on-the-fly tramite derivate parziali per Normal Mapping
    if (length(mapColor) > 0.1) {
        float factorNormal = 1.3;
        vec3 mapNormal = (mapColor * factorNormal) - 1.0;
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

    // --- 3. COMPONENTE DIFFUSA ---
    float bumpRoom = (max(dot(finalNormal, L_room), 0.0) * 0.2) + 1.0;
    float diffFlash = max(dot(finalNormal, L_flash), 0.0);

    // --- 4. RIFLESSO SPECULARE BLINN-PHONG ---
    // Il vettore di vista V punta dal frammento alla camera (origine)
    vec3 V = normalize(-ViewPos);
    float NdotH_room = max(dot(finalNormal, normalize(L_room + V)), 0.0);
    float NdotH_flash = max(dot(finalNormal, normalize(L_flash + V)), 0.0);

    // Gestione differenziata per Pavimento (Horizontal) vs Muri
    float isHorizontal = step(0.8, abs(finalNormal.y));
    float shininess = mix(16.0, 4.0, isHorizontal);
    float specBoost = mix(0.5, 1.5, isHorizontal);

    float specularRoom = clamp(pow(NdotH_room, shininess) * specBoost, 0.0, 1.0);
    float specularFlash = clamp(pow(NdotH_flash, shininess) * specBoost, 0.0, 1.0);

    // --- 5. DECADIMENTI ED EFFETTO BUIO ---

    // Decadimento Stanza (Inversamente proporzionale a LightIntensity)
    // Se LightIntensity tende a 0, il decadimento è massimo.
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float roomFalloff = exp(-FragDepth * decayRate * 0.1);

    // Attenuazione quadratica della torcia per non illuminare a distanza infinita
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    flashIntensity = flashFalloff * flashFalloff;

    // --- 6. FINAL MIX ---

    // Contributo Luce Settore
    vec3 litRoom = (texColor.rgb * bumpRoom * (0.3 + roomSpotIntensity * 1.7) + vec3(specularRoom * roomSpotIntensity)) * roomFalloff;

    // Contributo Torcia (Luce leggermente calda)
    vec3 flashColor = vec3(1.0, 0.98, 0.9);
    vec3 litFlash = (texColor.rgb * diffFlash + vec3(specularFlash)) * flashIntensity * flashColor;

    // Somma delle luci e correzione Gamma
    vec3 finalColor = pow(max(litRoom + litFlash, 0.0), vec3(0.8));

    FragColor = vec4(finalColor, texColor.a);
}