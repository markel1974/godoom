#version 330 core

in vec3 TexCoords;
in vec3 ViewPos;

layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_texture[4];

uniform sampler2D u_refractionTex;
uniform sampler2D u_depthTex;

uniform vec2 u_resolution;
uniform float u_time;

uniform float u_near;
uniform float u_far;

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);

    if (b == 0) return texture(u_texture[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_texture[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_texture[2], vec3(tc.xy, l));

    return texture(u_texture[3], vec3(tc.xy, l));
}

void main() {
    vec2 screenUV = gl_FragCoord.xy / u_resolution;
    // Water texture
    vec3 texColor = getDiffuse(TexCoords).rgb;
    // Animated style water distortion
    float lum = dot(texColor, vec3(0.299, 0.587, 0.114));
    vec2 wave1 = vec2(sin(TexCoords.x * 25.0 + u_time * 1.5), cos(TexCoords.y * 25.0 + u_time * 1.2));
    vec2 wave2 = vec2(cos(TexCoords.y * 40.0 - u_time * 1.1), sin(TexCoords.x * 40.0 - u_time * 1.4));
    vec2 distortion = (wave1 + wave2) * (0.002 + lum * 0.006);
    vec2 distortedUV = clamp(screenUV + distortion, 0.001, 0.999);
    // Refraction
    vec3 refractionColor = texture(u_refractionTex, distortedUV).rgb;
    // Background depth
    // IMPORTANT: depth is sampled using the original screen position,not the distorted refraction UV.
    float bgDepthRaw = texture(u_depthTex, screenUV).r;
    float zN = bgDepthRaw * 2.0 - 1.0;
    float bgDepth = (2.0 * u_near * u_far) / (u_far + u_near - zN * (u_far - u_near));

    // View-space Z of water surface.
    float waterDepth = abs(ViewPos.z);
    float depthDiff = max(bgDepth - waterDepth, 0.0);

    // Beer-Lambert absorption
    vec3 waterFogColor = vec3(0.0, 0.2, 0.3);
    float absorption = 0.02;
    float transmittance = exp(-depthDiff * absorption);
    vec3 finalColor = refractionColor * transmittance + waterFogColor * (1.0 - transmittance);
    // Mix in the actual water texture.
    finalColor = mix(finalColor, finalColor * texColor, 0.35);

    // Output
    FragColor = vec4(finalColor, 0.85);

    float brightness = dot(finalColor, vec3(0.2126, 0.7152, 0.0722));

    BrightColor = vec4(brightness > 3.0? finalColor: vec3(0.0), 1.0);
}
