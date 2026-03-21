#version 330 core
out float FragColor;

in vec2 TexCoords;

uniform sampler2D gPosition;
uniform sampler2D gNormal;
uniform sampler2D texNoise;

uniform vec3 samples[64];
uniform mat4 projection;

int kernelSize = 64;
float radius = 16.0;
float bias = 0.5;

void main() {
    vec3 fragPos = texture(gPosition, TexCoords).xyz;
    vec3 normal  = normalize(texture(gNormal, TexCoords).rgb);

    vec2 noiseScale = vec2(textureSize(gPosition, 0)) / 4.0;
    vec3 randomVec = normalize(texture(texNoise, TexCoords * noiseScale).rgb);

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
    for(int i = 0; i < kernelSize; ++i) {
        vec3 samplePos = TBN * samples[i];
        samplePos = fragPos + samplePos * radius;

        vec4 offset = vec4(samplePos, 1.0);
        offset = projection * offset;
        offset.xyz /= offset.w;
        offset.xyz = offset.xyz * 0.5 + 0.5;

        float sampleDepth = texture(gPosition, offset.xy).z;

        // 2. FIX ALONI NERI: Inversione logica del Range Check
        // Calcoliamo la distanza assoluta e usiamo un falloff decrescente da 1.0 a 0.0.
        // Se la differenza supera il radius, l'occlusione diventa un perfetto 0.0 senza sbavature.
        float depthDiff = abs(fragPos.z - sampleDepth);
        float rangeCheck = smoothstep(1.0, 0.0, depthDiff / radius);

        occlusion += (sampleDepth >= samplePos.z + bias ? 1.0 : 0.0) * rangeCheck;
    }

    FragColor = 1.0 - (occlusion / kernelSize);
}