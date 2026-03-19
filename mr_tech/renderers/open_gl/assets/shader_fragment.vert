#version 330 core

out vec4 FragColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;
in vec3 ViewPos;           // View Space
in vec3 LightCenterView;   // View Space
in vec3 NormalView;        // View Space

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform sampler2D u_ssao;           // Nuova: Texture SSAO generata dal pass precedente
uniform vec2 u_screenResolution;    // Nuova: Risoluzione per mappare gl_FragCoord
uniform bool u_hasNormalMap;
uniform float u_ambient_light;
uniform vec3 u_flashDir;

void main()
{
    // Campionamento lineare standard per MipMapping hardware nativo
    vec4 texColor = texture(u_texture, TexCoords);

    // Hard cutoff alpha per il billboarding
    if(texColor.a < 0.5) {
        discard;
    }

    // --- 0. RECUPERO OCCLUSIONE (SSAO) ---
    // Calcoliamo le coordinate schermo per campionare la texture SSAO
    vec2 ssaoCoords = gl_FragCoord.xy / u_screenResolution;
    float ao = texture(u_ssao, ssaoCoords).r;

    // --- 1. VETTORI DI ILLUMINAZIONE (View Space) ---

    // A. Luce del Settore (Zenitale)
    vec3 lightVectorRoom = LightCenterView - ViewPos;
    vec3 L_room = normalize(lightVectorRoom);

    // 1. Luce Diretta (Spotlight principale verso il pavimento)
    vec3 spotDirRoom = vec3(0.0, -1.0, 0.0);
    float cosThetaRoom = dot(-L_room, spotDirRoom);
    float directSpot = smoothstep(0.30, 0.50, cosThetaRoom);

    // 2. Luce Indiretta / Radiosity (Rimbalzo dal pavimento verso l'alto)
    vec3 bounceDir = vec3(0.0, 1.0, 0.0);
    float cosThetaBounce = dot(-L_room, bounceDir);
    // Il rimbalzo ha un cono molto più ampio (diffusione) e un'intensità ridotta (albedo)
    float bounceSpot = smoothstep(0.0, 0.80, cosThetaBounce) * 0.15; // 15% di riflessione

    float roomSpotIntensity = max(directSpot, bounceSpot);


    // B. Torcia (Allineata alla visuale Y-Sheared)
    vec3 L_flash = normalize(-ViewPos);
    vec3 viewFront = normalize(u_flashDir);
    float cosThetaFlash = dot(-L_flash, viewFront);
    float flashCone = smoothstep(0.85, 0.95, cosThetaFlash);

    // --- 2. NORMALE E BUMP MAPPING (TBN Analitico) ---
    vec3 finalNormal = NormalView;
    if (dot(finalNormal, finalNormal) < 0.01) {
        finalNormal = vec3(0.0, 1.0, 0.0);
    } else {
        finalNormal = normalize(finalNormal);
    }

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) > 0.1) {
        vec3 unpacked = (mapColor * 2.0) - 1.0;
        // Miscelazione per il controllo dell'intensità del bump
        vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

        // Derivate in screen-space di posizione e UV
        vec3 dp1 = dFdx(ViewPos);
        vec3 dp2 = dFdy(ViewPos);
        vec2 duv1 = dFdx(TexCoords);
        vec2 duv2 = dFdy(TexCoords);

        // Risoluzione del sistema lineare per allineare T e B alle UV
        vec3 dp2perp = cross(dp2, finalNormal);
        vec3 dp1perp = cross(finalNormal, dp1);
        vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
        vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

        // Costruzione e normalizzazione della matrice ortonormale
        float invmax = inversesqrt(max(dot(T, T), dot(B, B)));
        mat3 TBN = mat3(T * invmax, B * invmax, finalNormal);

        finalNormal = normalize(TBN * mapNormal);
    }

    // --- 3. COMPONENTE DIFFUSA ---
    float bumpRoom = (max(dot(finalNormal, L_room), 0.0) * 0.2) + 1.0;
    // Wrap Lighting per la torcia: impedisce l'azzeramento sulle superfici radenti
    float NdotL_flash = dot(finalNormal, L_flash);
    float diffFlash = max(NdotL_flash * 0.5 + 0.5, 0.0);

    // --- 4. RIFLESSO SPECULARE BLINN-PHONG ---
    vec3 V = normalize(-ViewPos);
    float NdotH_room = max(dot(finalNormal, normalize(L_room + V)), 0.0);
    float NdotH_flash = max(dot(finalNormal, normalize(L_flash + V)), 0.0);

    float isHorizontal = step(0.8, abs(finalNormal.y));
    float shininess = mix(16.0, 4.0, isHorizontal);
    float specBoost = mix(0.5, 1.5, isHorizontal);

    float specularRoom = clamp(pow(NdotH_room, shininess) * specBoost, 0.0, 1.0);
    float specularFlash = clamp(pow(NdotH_flash, shininess) * specBoost, 0.0, 1.0);

    // --- 5. DECADIMENTI ED EFFETTO BUIO ---
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float roomFalloff = exp(-FragDepth * decayRate * 0.1);

    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * flashFalloff * 10;

    // --- 6. FINAL MIX ---
    // Applichiamo 'ao' al termine ambientale (0.3) della luce di settore.
    // Questo scurisce gli angoli e le intersezioni dove la luce indiretta non arriva.
    vec3 litRoom = (texColor.rgb * bumpRoom * (0.3 * ao + roomSpotIntensity * 1.7) + vec3(specularRoom * roomSpotIntensity)) * roomFalloff;

    vec3 flashColor = vec3(1.0, 0.98, 0.9);
    // La torcia, essendo una luce diretta e puntiforme, solitamente non viene influenzata dal SSAO.
    vec3 litFlash = (texColor.rgb * diffFlash + vec3(specularFlash)) * flashIntensity * flashColor;

    vec3 linearColor = max(litRoom + litFlash, 0.0);

    // Ripristinata la curva gamma a 0.8 per la saturazione dei neri
    vec3 finalColor = pow(linearColor, vec3(0.8));

    // Dithering spaziale per il padding cromatico a 8-bit
    float dither = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) / 255.0;

    FragColor = vec4(finalColor + dither - (0.5 / 255.0), texColor.a);
}