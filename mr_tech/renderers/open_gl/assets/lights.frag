#version 330 core

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

in vec2 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec3 NormalView;
in vec4 FragPosLightRoom;

uniform sampler2D u_texture;
uniform sampler2D u_normalMap;
uniform sampler2DShadow u_roomShadowMap;
uniform mat4 u_view;
uniform mat4 u_invView;
uniform mat4 u_roomSpaceMatrix;

uniform vec2 u_screenResolution;
uniform float u_ambient_light;
uniform int u_enableShadows;
uniform int u_volumetricSteps;
uniform float u_beamRatioFactor;

layout(std140) uniform LightsBlock {
    vec4 u_lights[256];
    int u_numLights;
};

float randomNoise(vec2 co) {
    return fract(sin(dot(co, vec2(12.9898, 78.233))) * 43758.5453);
}

float sampleVolumetricShadow(vec3 posView, mat4 lightSpaceMatrix, sampler2DShadow shadowMap) {
    vec4 worldPos = u_invView * vec4(posView, 1.0);
    vec4 shadowPos = lightSpaceMatrix * worldPos;
    vec3 proj = shadowPos.xyz / shadowPos.w;
    proj = proj * 0.5 + 0.5;
    if(proj.z > 1.0 || proj.x < 0.0 || proj.x > 1.0 || proj.y < 0.0 || proj.y > 1.0) return 1.0;
    return texture(shadowMap, vec3(proj.xy, proj.z - 0.005));
}

float shadowCalculation(vec4 fragPosLightSpace, sampler2DShadow shadowMap, float bias) {
    if (fragPosLightSpace.w <= 0.0) return 0.0;
    vec3 projCoords = fragPosLightSpace.xyz / fragPosLightSpace.w;
    projCoords = projCoords * 0.5 + 0.5;
    if(projCoords.z > 1.0 || projCoords.x < 0.0 || projCoords.x > 1.0 || projCoords.y < 0.0 || projCoords.y > 1.0) return 0.0;

    float currentDepth = projCoords.z;
    float shadow = 0.0;
    vec2 texelSize = 1.0 / vec2(textureSize(shadowMap, 0));
    const int SAMPLES = 16;
    const float GOLDEN_ANGLE = 2.39996323;
    float noise = randomNoise(gl_FragCoord.xy) * 6.2831853;
    float spread = 2.0;

    for(int i = 0; i < SAMPLES; ++i) {
        float r = sqrt(float(i) + 0.5) / sqrt(float(SAMPLES));
        float theta = float(i) * GOLDEN_ANGLE + noise;
        vec2 offset = vec2(cos(theta), sin(theta)) * r * spread;
        shadow += texture(shadowMap, vec3(projCoords.xy + offset * texelSize, currentDepth - bias));
    }
    return 1.0 - (shadow / float(SAMPLES));
}

vec3 calculateNormal(vec3 baseNormal) {
    // Rimosso il gl_FrontFacing: rispetta la normale geometrica passata dal Vertex Shader
    if (dot(baseNormal, baseNormal) < 0.01) return vec3(0.0, 1.0, 0.0);
    vec3 normal = normalize(baseNormal);

    vec3 mapColor = texture(u_normalMap, TexCoords).rgb;
    if (length(mapColor) < 0.1) return normal;

    vec3 unpacked = (mapColor * 2.0) - 1.0;
    vec3 mapNormal = normalize(mix(vec3(0.0, 0.0, 1.0), unpacked, 0.7));

    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);
    vec2 duv1 = dFdx(TexCoords);
    vec2 duv2 = dFdy(TexCoords);

    vec3 dp2perp = cross(dp2, normal);
    vec3 dp1perp = cross(normal, dp1);
    vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
    vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

    float denom = max(dot(T, T), dot(B, B));

    // TRAPPOLA HARDWARE (Anti-NaN)
    // Se dFdx esplode a Infinito sui triangoli degeneri, denom va a Inf.
    // inversesqrt(Inf) fa 0. Inf * 0 fa NaN!
    // Il limite < 1e6 blocca gli infiniti e salva il frammento.
    if (denom > 1e-5 && denom < 1e6) {
        float invmax = inversesqrt(denom);
        mat3 TBN = mat3(T * invmax, B * invmax, normal);
        return normalize(TBN * mapNormal);
    }

    return normal;
}

void main()
{
    vec4 texColor = texture(u_texture, TexCoords);
    if(texColor.a < 0.5) discard;

    vec3 albedo = pow(texColor.rgb, vec3(2.2));
    vec2 screenUV = gl_FragCoord.xy / u_screenResolution;
    float edgeFade = smoothstep(0.0, 0.08, screenUV.x) * smoothstep(1.0, 0.92, screenUV.x);

    vec3 finalNormal = calculateNormal(NormalView);
    vec3 L_room_dir = normalize(mat3(u_view) * vec3(0.0, 1.0, 0.0));

    // OMBRE STANZA
    float shadowRoom = 0.0;
    if (u_enableShadows == 1) {
        vec3 geoNormal = normalize(NormalView);
        float roomBias = max(0.0005 * (1.0 - clamp(dot(geoNormal, L_room_dir), 0.0, 1.0)), 0.0001);
        shadowRoom = shadowCalculation(FragPosLightRoom, u_roomShadowMap, roomBias);
    }

    // 1. LUCE AMBIENTALE DIREZIONALE (Risolve il problema del "troppo chiaro")
    float NdotL_room = max(dot(finalNormal, L_room_dir), 0.0);
    float shadowFactor = (0.3 + 0.7 * (1.0 - shadowRoom));
    vec3 litRoom = albedo * NdotL_room * u_ambient_light * shadowFactor;
    // 2. NEBBIA VOLUMETRICA AMBIENTALE
    float volRoom = 0.0;
    vec3 rayStep = ViewPos / float(u_volumetricSteps);
    vec3 currentPos = rayStep * randomNoise(gl_FragCoord.xy);
    for(int i = 0; i < u_volumetricSteps * 2; i++) {
        float fogGlow = exp(-length(currentPos) * 0.005) * u_ambient_light;
        float sRoom = sampleVolumetricShadow(currentPos, u_roomSpaceMatrix, u_roomShadowMap);
        volRoom += fogGlow * sRoom * 0.15;
        currentPos += rayStep;
    }
    vec3 roomBeam = vec3(1.0, 0.95, 0.85) * volRoom * (u_beamRatioFactor / float(u_volumetricSteps)) * edgeFade;

    // 3. LUCI DINAMICHE (UBO)
    vec3 dynamicLights = vec3(0.0);

    vec3 spotDirView = normalize(mat3(u_view) * vec3(0.0, -1.0, 0.0));

    // 2. Definisci l'apertura del cono in gradi (usiamo il coseno perché dot product restituisce il coseno)
    float cutOff = cos(radians(25.0));       // Il cerchio di luce piena centrale
    float outerCutOff = cos(radians(35.0));  // La sfumatura esterna del cono
    int debug = 0;

    for (int i = 0; i < u_numLights; ++i) {
        float normalizedIntensity = u_lights[i].w;
        if (normalizedIntensity <= 0.001) continue;
        if (normalizedIntensity > 1) {
            normalizedIntensity = 1;
        }

        vec3 lightPosView = (u_view * vec4(u_lights[i].xyz, 1.0)).xyz;
        vec3 L = lightPosView - ViewPos;
        float dist = length(L);
        L = L / dist; // L punta dal frammento alla luce

        // Calcolo Lambertiano puro
        float NdotL = max(dot(finalNormal, L), 0.0);

        // --- HACK VOLUMETRICO ---
        // Forza un'illuminazione di base (es. 40%) per qualsiasi geometria
        // che si trovi fisicamente all'interno del cono, bypassando la normale.
        float wrapLight = max(NdotL, 0.4);

        // --- LOGICA DEL CONO ---
        vec3 lightToFrag = -L;
        float theta = dot(lightToFrag, spotDirView);
        float spotEffect = smoothstep(outerCutOff, cutOff, theta);

        float power = normalizedIntensity * 30.0;
        float fallofFactor = 200.0;//150.0;
        float falloff = exp(-dist / (fallofFactor * max(normalizedIntensity, 0.1)));
        //float falloff = 0.1;
        dynamicLights += albedo * wrapLight * power * falloff * spotEffect;

        if (debug == 1) {
            // --- DEBUG ORIGINE LUCE ---
            // Vettore normalizzato dalla telecamera al frammento attuale
            vec3 viewRay = normalize(ViewPos);
            // Distanza perpendicolare tra l'origine della luce e il raggio visivo
            float distToRay = length(cross(lightPosView, viewRay));
            float depthToLight = length(lightPosView);
            float depthToFrag = length(ViewPos);
            // Se il raggio incrocia il volume della luce (raggio 4.0 unità) e la luce NON è dietro un muro
            if (distToRay < 1.0 && depthToLight < depthToFrag) {
                float bulb = smoothstep(1.0, 0.0, distToRay);
                // Disegna un nucleo magenta brillante per identificare l'origine esatta
                dynamicLights += vec3(1.0, 0.0, 1.0) * bulb * 10.0;
            }
        }
    }

    vec3 finalLight = litRoom + roomBeam + dynamicLights;


    /*
    vec3 debugFootprint = vec3(0.0);
    for (int i = 0; i < u_numLights; ++i) {
        float intensity = u_lights[i].w;
        if (intensity <= 0.001) continue;
        vec3 lightPosView = (u_view * vec4(u_lights[i].xyz, 1.0)).xyz;
        vec3 L = lightPosView - ViewPos;
        float dist = length(L);
        L = normalize(L);
        float NdotL = max(dot(finalNormal, L), 0.0);
        float falloff = exp(-dist * 0.015);
        // Macchia solida, senza anelli
        debugFootprint += vec3(1.0, 0.0, 0.0) * NdotL * intensity * falloff;
    }
    FragColor = vec4(vec3(0.05) + debugFootprint, 1.0);
    */

    // albedo = vec3(0.8);
    // NdotL_room = 0.5;
    // shadowFactor = 0.5;
    // ambientLight = 0.8;
    // roomBeam = vec3(0.0);
    // dynamicLights = vec3(0.4, 0.4, 0.4);

    FragColor = vec4(finalLight, 0.0);
    BrightColor = vec4(dot(finalLight, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalLight : vec3(0.0), 1.0);
}