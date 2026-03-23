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
uniform float u_flashBase;
uniform int u_enableShadows;
uniform sampler2D u_emissiveMap;

uniform float u_emissiveIntensity;
uniform float u_shininessWall;
uniform float u_shininessFloor;
uniform float u_specBoostWall;
uniform float u_specBoostFloor;
uniform float u_beamRatioFactor;
uniform float u_aoFactor;
uniform float u_roomSpotIntensityFactor;
uniform int u_volumetricSteps;

float ShadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;

    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;

    if(projCoords.z > 1.0) return 0.0;

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));

    // Sottrazione del bias dinamico per abbattere il self-shadowing di prossimità
    for(int x = -1; x <= 1; ++x) {
        for(int y = -1; y <= 1; ++y) {
            shadow += texture(shadowMap, vec3(projCoords.xy + vec2(x, y) * texelSize * 1.5, currentDepth - bias));
        }
    }

    return 1.0 - (shadow / 9.0);
}

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    // FIX: Converti da sRGB a Spazio Lineare
    vec3 albedo = pow(texColor.rgb, vec3(2.2));

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
        // 1. Bias lineare per proiezione Ortografica (la luce viaggia lungo spotDirRoom)
        // Il vettore verso la luce è -spotDirRoom
        float ndotlRoom = clamp(dot(geoNormal, -spotDirRoom), 0.0, 1.0);
        float roomBias = max(0.002 * (1.0 - ndotlRoom), 0.0005);
        // 2. Bias logaritmico per proiezione Prospettica (la luce proviene da L_flash)
        float ndotlFlash = clamp(dot(geoNormal, L_flash), 0.0, 1.0);
        float flashBias = max(0.002 * (1.0 - ndotlFlash), 0.0005);
        shadowRoom = ShadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
        shadowFlash = ShadowCalculation(FragPosLightFlash, u_flashShadowMap, flashBias);
    }

    // --- ILLUMINAZIONE E RIFLESSI ---
    float bumpRoom = (max(dot(finalNormal, L_room_point), 0.0) * 0.2) + 1.0;
    float diffFlash = max((dot(finalNormal, L_flash) * 0.5) + u_flashBase, 0.0);

    vec3 V = normalize(-ViewPos);
    vec3 H_room = L_room_point + V;
    float NdotH_room = length(H_room) > 0.0001 ? max(dot(finalNormal, normalize(H_room)), 0.0) : 0.0;

    vec3 H_flash = L_flash + V;
    float NdotH_flash = length(H_flash) > 0.0001 ? max(dot(finalNormal, normalize(H_flash)), 0.0) : 0.0;

    float isHorizontal = step(0.8, abs(finalNormal.y));
    float shininess = mix(u_shininessWall, u_shininessFloor, isHorizontal);
    float specBoost = mix(u_specBoostWall, u_specBoostFloor, isHorizontal);
    float specularRoom = clamp(pow(NdotH_room, shininess) * specBoost, 0.0, 1.0);
    float specularFlash = clamp(pow(NdotH_flash, shininess) * specBoost, 0.0, 1.0);

    float decayRate = (LightDist >= 0.0) ? LightDist : u_ambient_light;
    float roomFalloff = exp(-FragDepth * decayRate * 0.1);
    float flashFalloff = 1.0 / (1.0 + (0.05 * FragDepth) + 0.005 * (FragDepth * FragDepth));
    float flashIntensity = flashCone * (flashFalloff * u_flashIntensityFactor);

    // --- RAYMARCHING VOLUMETRICO ---
    float volumetricScattering = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * 0.5;
    for(int i = 0; i < u_volumetricSteps; i++) {
        vec3 toLight = LightCenterView - currentPos;
        vec3 lDir = normalize(toLight);
        float cosTheta = dot(-lDir, spotDirRoom);
        float inCone = smoothstep(0.30, 0.50, cosTheta);
        volumetricScattering += inCone;
        currentPos += rayStep;
    }

    float beamRatio = volumetricScattering / float(u_volumetricSteps);
    // APPLICA L'EDGE FADE AL RAGGIO: Evita che il fascio di luce si tronchi di netto se il muro dietro viene scartato
    vec3 beamColor = vec3(1.0, 0.95, 0.85) * (beamRatio * u_beamRatioFactor) * roomFalloff * edgeFade;

    // --- MIX FINALE ---
    float roomLightOcclusion = (1.0 - shadowRoom);
    float flashLightOcclusion = (1.0 - shadowFlash);

    vec3 litRoom = (albedo * bumpRoom * ((ao * u_aoFactor) + (roomSpotIntensity * u_roomSpotIntensityFactor * roomLightOcclusion)) + vec3(specularRoom * roomSpotIntensity * roomLightOcclusion)) * roomFalloff;

    vec3 flashColor = vec3(1.0, 0.98, 0.9);
    vec3 litFlash = (albedo * diffFlash + vec3(specularFlash)) * flashIntensity * flashLightOcclusion * flashColor;

    vec3 emissive = texture(u_emissiveMap, TexCoords).rgb;

    vec3 linearColor = max(litRoom + litFlash, 0.0);
    linearColor += beamColor;

    // APPLICA L'EDGE FADE ALL'EMISSIVE: Impedisce al Bloom del neon/fuoco di "poppare" quando lo sprite esce dallo schermo
    linearColor += (emissive * u_emissiveIntensity) * edgeFade;

    FragColor = vec4(linearColor, texColor.a);
    float brightness = dot(linearColor, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(brightness > 3.0 ? linearColor : vec3(0.0), 1.0);
}