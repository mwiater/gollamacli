config.Personas.json: "Explain the best way to get a cat down from a very tall tree."
config.Authors.json: "Describe a brief argument over a minor misunderstanding in a public place."
config.CognitiveConstraints.json: "Explain the concept of 'biting the bullet' and give an example."

Tell me how to cook a hamburger in a cast iron skillet.

goreleaser release --snapshot --clean --skip archive

clear && ./dist/gollamacli_windows_amd64_v1/gollamacli.exe chat --config config.Personas.json
clear && ./dist/gollamacli_windows_amd64_v1/gollamacli.exe chat --config config.Authors.json
clear && ./dist/gollamacli_windows_amd64_v1/gollamacli.exe chat --config config.CognitiveConstraints.json
