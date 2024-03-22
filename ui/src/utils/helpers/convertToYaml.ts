import yaml from 'js-yaml';

export function convertToYaml(data: any) {
    return yaml.dump(data)
}