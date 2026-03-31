#version 330 core
out float FragColor;

in vec3 TexCoords;

uniform sampler2D u_position;
uniform sampler2D u_normal;
uniform sampler2D u_texNoise;

uniform vec3 u_samples[64];
uniform mat4 u_projection;

uniform int u_kernelSize;
uniform float u_radius;
uniform float u_bias;

void main() {
    vec3 fragPos = texture(u_position, TexCoords).xyz;
    vec3 normal  = normalize(texture(u_normal, TexCoords).rgb);

    vec2 noiseScale = vec2(textureSize(u_position, 0)) / 4.0;
    vec3 randomVec = normalize(texture(u_texNoise, TexCoords * noiseScale).rgb);

    // 1. PROTEZIONE ANTI-NaN: Evita i buchi neri geometrici se i vettori sono paralleli
    vec3 tangent = randomVec - normal * dot(randomVec, normal);
    if (length(tangent) > 0.001) {
        tangent = normalize(tangent);
    } else {
        tangent = normalize(cross(normal, vec3(0.0, 1.0, 0.0)));
        if (length(tangent) < 0.001) {
            tangent = normalize(cross(normal, vec3(1.0, 0.0, 0.0)));
        }
    }

    vec3 bitangent = cross(normal, tangent);
    mat3 TBN       = mat3(tangent, bitangent, normal);

    float occlusion = 0.0;
    for(int i = 0; i < u_kernelSize; ++i) {
        vec3 samplePos = TBN * u_samples[i];
        samplePos = fragPos + samplePos * u_radius;

        vec4 offset = vec4(samplePos, 1.0);
        offset = u_projection * offset;
        offset.xyz /= offset.w;
        offset.xyz = offset.xyz * 0.5 + 0.5;

        float sampleDepth = texture(u_position, offset.xy).z;

        // 2. FIX ALONI NERI: Inversione logica del Range Check
        // Calcoliamo la distanza assoluta e usiamo un falloff decrescente da 1.0 a 0.0.
        // Se la differenza supera il radius, l'occlusione diventa un perfetto 0.0 senza sbavature.
        float depthDiff = abs(fragPos.z - sampleDepth);
        float rangeCheck = smoothstep(1.0, 0.0, depthDiff / u_radius);

        occlusion += (sampleDepth >= samplePos.z + u_bias ? 1.0 : 0.0) * rangeCheck;
    }

    FragColor = 1.0 - (occlusion / u_kernelSize);
}