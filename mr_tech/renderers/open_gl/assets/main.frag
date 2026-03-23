#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 TexCoords;
in float LightDist;
in float FragDepth;
in vec3 ViewPos;
in vec3 LightCenterView;
in vec3 NormalView;

in vec4 FragPosLightRoom;
in vec4 FragPosLightFlash;

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform sampler2D u_ssao;
uniform sampler2DShadow u_roomShadowMap;
uniform sampler2DShadow u_flashShadowMap;

uniform mat4 u_view;

uniform vec2 u_screenResolution;
uniform float u_ambient_light;
uniform vec3 u_flashDir;
uniform float u_flashIntensityFactor;
uniform vec3 u_flashOffset;
uniform float u_flashConeStart;
uniform float u_flashConeEnd;
uniform int u_enableShadows;
uniform sampler2D u_emissiveMap;

// --- Pura Proiezione Vettoriale HW ---
// Rimosso normal e lightDirViewSpace, non servono per il test di profondità nativo.
// --- FUNZIONE SHADOW MAPPING ---
// Reintroduciamo normal e lightDir per calcolare un micro-bias adattivo
float ShadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, vec3 normal, vec3 lightDir) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;

    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;

    if(projCoords.z > 1.0) {
        return 0.0;
    }

    // Micro-bias adattivo: più la luce è parallela al muro, maggiore è la protezione necessaria
    float ndotl = clamp(dot(normal, lightDir), 0.0, 1.0);
    float bias = max(0.005 * (1.0 - ndotl), 0.0005);

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));

    // PCF Bilineare
    for(int x = -1; x <= 1; ++x) {
        for(int y = -1; y <= 1; ++y) {
            // Sottraiamo il bias per evitare che i sample adiacenti "sbattano" contro il muro
            shadow += texture(shadowMap, vec3(projCoords.xy + vec2(x, y) * texelSize * 1.5, currentDepth - bias));
        }
    }

    return 1.0 - (shadow / 9.0);
}

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    // --- EDGE FADE CINEMATICO ---
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    // Sfuma dolcemente la luce nel primo e nell'ultimo 8% dello schermo orizzontale
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    vec2 ssaoCoords = gl_FragCoord.xy / u_screenResolution;
    float ao = texture(u_ssao, ssaoCoords).r;

    // --- VETTORI DI ILLUMINAZIONE ---
    vec3 L_room_point = normalize(LightCenterView - ViewPos);
    vec3 spotDirRoom = normalize(mat3(u_view) * vec3(0.0, -1.0, 0.0));

    float cosThetaRoom = dot(-L_room_point, spotDirRoom);
    float directSpot = smoothstep(0.30, 0.50, cosThetaRoom);
    vec3 bounceDir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));
    float cosThetaBounce = dot(-L_room_point, bounceDir);
    float bounceSpot = smoothstep(0.0, 0.80, cosThetaBounce) * 0.15;

    float roomSpotIntensity = max(directSpot, bounceSpot);

    // Torcia: Uso dell'Uniform Parametrica
    vec3 flashPosView = u_flashOffset;
    vec3 L_flash = normalize(flashPosView - ViewPos);

    vec3 viewFront = normalize(u_flashDir);
    float cosThetaFlash = dot(-L_flash, viewFront);
    float flashCone = smoothstep(u_flashConeStart, u_flashConeEnd, cosThetaFlash);

    // --- NORMALI E GUARDIE ANTI-NaN ---
    vec3 finalNormal = NormalView;
    if (dot(finalNormal, finalNormal) < 0.01) {
        finalNormal = vec3(0.0, 1.0, 0.0);
    } else {
        finalNormal = normalize(finalNormal);
    }

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) > 0.1) {
        vec3 unpacked = (mapColor * 2.0) - 1.0;
        vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

        vec3 dp1 = dFdx(ViewPos);
        vec3 dp2 = dFdy(ViewPos);
        vec2 duv1 = dFdx(TexCoords);
        vec2 duv2 = dFdy(TexCoords);

        vec3 dp2perp = cross(dp2, finalNormal);
        vec3 dp1perp = cross(finalNormal, dp1);
        vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
        vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

        float denom = max(dot(T, T), dot(B, B));
        if (denom > 1e-5) {
            float invmax = inversesqrt(denom);
            mat3 TBN = mat3(T * invmax, B * invmax, finalNormal);
            finalNormal = normalize(TBN * mapNormal);
        }
    }

    // --- CALCOLO OMBRE ---
    float shadowRoom = 0.0;
    float shadowFlash = 0.0;
    // Usiamo la normale geometrica nuda per un bias stabile senza subire i bump della normal map

    if (u_enableShadows == 1) {
        vec3 geoNormal = normalize(NormalView);
        shadowRoom = ShadowCalculation(FragPosLightRoom, u_roomShadowMap, geoNormal, L_room_point);
        shadowFlash = ShadowCalculation(FragPosLightFlash, u_flashShadowMap, geoNormal, L_flash);
    }

    // --- ILLUMINAZIONE E RIFLESSI ---
    float bumpRoom = (max(dot(finalNormal, L_room_point), 0.0) * 0.2) + 1.0;
    float diffFlash = max(dot(finalNormal, L_flash) * 0.5 + 0.5, 0.0);

    vec3 V = normalize(-ViewPos);
    vec3 H_room = L_room_point + V;
    float NdotH_room = length(H_room) > 0.0001 ? max(dot(finalNormal, normalize(H_room)), 0.0) : 0.0;

    vec3 H_flash = L_flash + V;
    float NdotH_flash = length(H_flash) > 0.0001 ? max(dot(finalNormal, normalize(H_flash)), 0.0) : 0.0;

    float isHorizontal = step(0.8, abs(finalNormal.y));
    float shininess = mix(64.0, 48.0, isHorizontal);
    float specBoost = mix(0.6, 0.8, isHorizontal);
    float specularRoom = clamp(pow(NdotH_room, shininess) * specBoost, 0.0, 1.0);
    float specularFlash = clamp(pow(NdotH_flash, shininess) * specBoost, 0.0, 1.0);
    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float roomFalloff = exp(-FragDepth * decayRate * 0.1);
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    // --- RAYMARCHING VOLUMETRICO ---
    float volumetricScattering = 0.0;
    const int STEPS = 16;
    vec3 rayStep = ViewPos / float(STEPS);
    vec3 currentPos = rayStep * 0.5;
    for(int i = 0; i < STEPS; i++) {
        vec3 toLight = LightCenterView - currentPos;
        vec3 lDir = normalize(toLight);
        float cosTheta = dot(-lDir, spotDirRoom);
        float inCone = smoothstep(0.30, 0.50, cosTheta);
        volumetricScattering += inCone;
        currentPos += rayStep;
    }

    float beamRatio = volumetricScattering / float(STEPS);
    float beamRatioFactor = 0.05;
    // APPLICA L'EDGE FADE AL RAGGIO: Evita che il fascio di luce si tronchi di netto se il muro dietro viene scartato
    vec3 beamColor = vec3(1.0, 0.95, 0.85) * (beamRatio * beamRatioFactor) * roomFalloff * edgeFade;

    // --- MIX FINALE ---
    float aoFactor = 0.3;
    float roomSpotIntensityFactor = 1.7;
    float roomLightOcclusion = (1.0 - shadowRoom);
    float flashLightOcclusion = (1.0 - shadowFlash);

    vec3 litRoom = (texColor.rgb * bumpRoom * ((ao * aoFactor) + (roomSpotIntensity * roomSpotIntensityFactor * roomLightOcclusion)) + vec3(specularRoom * roomSpotIntensity * roomLightOcclusion)) * roomFalloff;

    vec3 flashColor = vec3(1.0, 0.98, 0.9);
    vec3 litFlash = (texColor.rgb * diffFlash + vec3(specularFlash)) * flashIntensity * flashLightOcclusion * flashColor;

    vec3 emissive = texture(u_emissiveMap, TexCoords).rgb;
    float emissiveIntensity = 15.0;

    vec3 linearColor = max(litRoom + litFlash, 0.0);
    linearColor += beamColor;

    // APPLICA L'EDGE FADE ALL'EMISSIVE: Impedisce al Bloom del neon/fuoco di "poppare" quando lo sprite esce dallo schermo
    linearColor += (emissive * emissiveIntensity) * edgeFade;

    FragColor = vec4(linearColor, texColor.a);
    float brightness = dot(linearColor, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(brightness > 1.0 ? linearColor : vec3(0.0), 1.0);
}