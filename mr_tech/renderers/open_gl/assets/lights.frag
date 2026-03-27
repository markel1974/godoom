#version 330 core
out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D gAlbedoSpec;
uniform sampler2D gNormalEmiss;
uniform sampler2D gPositionDepth;

uniform mat4 u_view;
uniform mat4 u_invView;
uniform float u_ambient_light;

// UBO per le luci dinamiche
layout(std140) uniform LightsBlock {
    vec4 u_lights[256]; // xyz: Posizione World, w: Intensità
};
uniform int u_numLights;

void main() {
    vec4 albedoSpec = texture(gAlbedoSpec, TexCoords);
    vec4 normalEmiss = texture(gNormalEmiss, TexCoords);
    vec4 posDepth = texture(gPositionDepth, TexCoords);

    if (length(normalEmiss.rgb) < 0.1) discard; // Sfondo/Skybox

    vec3 albedo = albedoSpec.rgb;
    vec3 normal = normalize(normalEmiss.rgb);
    vec3 viewPos = posDepth.xyz;
    float emissive = normalEmiss.a;

    vec3 V = normalize(-viewPos);
    vec3 totalLight = albedo * u_ambient_light;

    // Calcolo radianza puntuale
    for (int i = 0; i < u_numLights; ++i) {
        float intensity = u_lights[i].w;
        if (intensity <= 0.001) continue;

        vec3 lightPosView = (u_view * vec4(u_lights[i].xyz, 1.0)).xyz;
        vec3 L = lightPosView - viewPos;
        float dist = length(L);
        L = L / dist;

        float falloff = exp(-dist * 0.015);
        float NdotL = max(dot(normal, L), 0.0);

        totalLight += albedo * NdotL * intensity * falloff;
    }

    totalLight += albedo * emissive;
    FragColor = vec4(totalLight, 1.0);
}