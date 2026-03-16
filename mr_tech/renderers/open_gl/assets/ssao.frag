#version 330 core
out float FragColor;

in vec2 TexCoords;

uniform sampler2D gPosition; // Texture contenente ViewPos.xyz
uniform sampler2D gNormal;   // Texture contenente NormalView.xyz
uniform sampler2D texNoise;  // Texture 4x4 di rumore per rotazione kernel

uniform vec3 samples[64];    // Kernel di campionamento (generato in Go)
uniform mat4 projection;     // Matrice di proiezione

int kernelSize = 64;
float radius = 16.0; // Deve essere comparabile alla scala spaziale di un gradino/spigolo nel modello
float bias = 0.5;    // Deve scalare linearmente con il radius per eliminare il self-shadowing planare

void main() {
    vec3 fragPos = texture(gPosition, TexCoords).xyz;
    vec3 normal  = normalize(texture(gNormal, TexCoords).rgb);
    // Ottiene la risoluzione dinamicamente dal livello di mip 0 del G-Buffer
    vec2 noiseScale = vec2(textureSize(gPosition, 0)) / 4.0;
    vec3 randomVec = normalize(texture(texNoise, TexCoords * noiseScale).rgb);

    // Creazione TBN per orientare l'emisfero
    vec3 tangent   = normalize(randomVec - normal * dot(randomVec, normal));
    vec3 bitangent = cross(normal, tangent);
    mat3 TBN       = mat3(tangent, bitangent, normal);

    float occlusion = 0.0;
    for(int i = 0; i < kernelSize; ++i) {
        vec3 samplePos = TBN * samples[i];
        samplePos = fragPos + samplePos * radius;

        // Proiezione del campione
        vec4 offset = vec4(samplePos, 1.0);
        offset = projection * offset;
        offset.xyz /= offset.w;
        offset.xyz = offset.xyz * 0.5 + 0.5;

        float sampleDepth = texture(gPosition, offset.xy).z;
        float rangeCheck = smoothstep(0.0, 1.0, radius / abs(fragPos.z - sampleDepth));
        occlusion += (sampleDepth >= samplePos.z + bias ? 1.0 : 0.0) * rangeCheck;
    }
    FragColor = 1.0 - (occlusion / kernelSize);
}