#version 330 core

in vec3 TexCoords;
in float FragDepth;
in vec3 ViewPos;
in vec4 FragPosLightRoom;
in vec4 FragPosLightFlash;
in float IsFullbright;

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_texture[4];

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_texture[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_texture[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_texture[2], vec3(tc.xy, l));
    return texture(u_texture[3], vec3(tc.xy, l));
}        // Normal map (water ripples)
uniform sampler2D u_refractionTex;  // Opaque Screen Color
uniform sampler2D u_depthTex;       // Opaque Screen Depth
uniform vec2 u_resolution;
uniform float u_time;

void main() {
    // Coordinate schermo per pescare la Refraction
    vec2 screenUV = gl_FragCoord.xy / u_resolution;

    // Normal Map (campionata con TexCoords.xy animato in vertex shader)
    vec4 normalData = getDiffuse(TexCoords);
    vec3 N = normalize(normalData.xyz * 2.0 - 1.0);
    
    // Distorsione Screen UV
    // Applichiamo la normale per creare l'effetto "onde" sullo sfondo.
    float distortionStrength = 0.05;
    vec2 distortedUV = screenUV + (N.xy * distortionStrength);
    
    // Clamping per non pescare fuori dallo schermo
    distortedUV = clamp(distortedUV, 0.001, 0.999);

    // Colore rifratto
    vec3 refractionColor = texture(u_refractionTex, distortedUV).rgb;

    // CALCOLO PROFONDITA (DEPTH FOG)
    // Leggiamo la profondità del frammento di sfondo (pavimento/muro)
    float bgDepthRaw = texture(u_depthTex, distortedUV).r;
    
    // Linearizziamo il Depth buffer (assumendo gl_FragCoord.z è non lineare)
    float near = 0.1;
    float far = 1000.0;
    // Standard perspective depth linearization
    float z_n = 2.0 * bgDepthRaw - 1.0;
    float bgDepth = 2.0 * near * far / (far + near - z_n * (far - near));
    
    // Profondità della superficie dell'acqua
    float waterDepth = abs(ViewPos.z);
    
    // Distanza effettiva sott'acqua
    float depthDiff = bgDepth - waterDepth;
    if (depthDiff < 0.0) {
        depthDiff = 0.0;
    }
    
    // Fog color (Assorbimento acqua)
    vec3 waterFogColor = vec3(0.0, 0.2, 0.3); // Colore acqua standard (Q1 slime potrebbe essere verde)
    
    // Se la texture ha un colore dominante lo usiamo (il diffuse color in Quake 1 è spesso melma o acqua blu)
    vec3 texColor = getDiffuse(TexCoords).rgb; // In Quake 1 questa spesso è Diffuse e non Normal.
    // Wait, in Q1 water textures are Diffuse! If we use them as normal maps, they'll look weird.
    // For now we will just use the texture as diffuse and pretend it's a normal map for ripples by taking luminance.
    
    float lum = dot(texColor, vec3(0.299, 0.587, 0.114));
    vec2 ripple = vec2(lum * 0.05);
    
    vec2 q1DistortedUV = screenUV + ripple;
    vec3 q1Refraction = texture(u_refractionTex, q1DistortedUV).rgb;
    
    // Uniamo la texture dell'acqua con quello che c'è sotto
    // Sfumatura basata sulla profondità (Beer's Law)
    float fogFactor = exp(-depthDiff * 0.02);
    
    vec3 finalColor = mix(texColor * 0.5, q1Refraction, fogFactor);

    FragColor = vec4(finalColor, 0.85); // 85% opaco (aggiunge trasparenza base)
    BrightColor = vec4(dot(finalColor, vec3(0.2126, 0.7152, 0.0722)) > 3.0 ? finalColor : vec3(0.0), 1.0);
}
