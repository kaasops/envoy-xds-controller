# План рефакторинга архитектуры ResBuilder v2

## Общая информация

Данный документ содержит план рефакторинга архитектуры пакета `internal/xds/resbuilder_v2`, направленный на улучшение модульности, тестируемости и поддерживаемости кода. Основная цель - разделение большого файла `builder.go` (~1000 строк) на логические компоненты.

## Проблемы текущей архитектуры

1. Файл `builder.go` содержит около 1000 строк кода с разнообразной функциональностью
2. Смешение различных уровней абстракции в одном файле
3. Высокая цикломатическая сложность некоторых функций
4. Затруднено тестирование отдельных компонентов
5. Код сложно поддерживать и расширять

## Компоненты для выделения

На основе анализа кода `builder.go` выделены следующие логические компоненты:

1. **HTTP Filters Management**:
   - `httpFiltersCache` и методы
   - `generateHTTPFiltersCacheKey`
   - `buildHTTPFilters`
   - `buildRBACFilter`

2. **Filter Chains Building**:
   - `FilterChainsParams`
   - `buildFilterChainParams`
   - `buildFilterChains`
   - `buildFilterChain`
   - `checkFilterChainsConflicts`

3. **Routing Configuration**:
   - `buildRouteConfiguration`
   - `buildVirtualHost`
   - `buildUpgradeConfigs`

4. **Access Logging**:
   - `buildAccessLogConfigs`

5. **TLS/Security Configuration**:
   - `getTLSType`
   - `getSecretNameToDomainsViaSecretRef`
   - `getSecretNameToDomainsViaAutoDiscovery`

6. **Main Resource Building**:
   - `ResourceBuilder` и методы
   - `Resources`
   - `BuildResources`
   - `buildResourcesFromVirtualService`
   - `buildResourcesFromExistingFilterChains`

7. **Utility Functions**:
   - `findClusterNames`
   - `findSDSNames`
   - `isTLSListener`
   - `checkAllDomainsUnique`
   - `getWildcardDomain`

## Новая структура пакетов

```
internal/xds/resbuilder_v2/
├── builder.go                   // Основной координатор (значительно уменьшенный)
├── resources.go                 // Определение структуры Resources
├── http_filters/
│   ├── cache.go                 // HTTP фильтр кэш и его методы
│   ├── builder.go               // Построение HTTP фильтров
│   └── rbac.go                  // Построение RBAC фильтров
├── filter_chains/
│   ├── params.go                // Структура FilterChainsParams
│   ├── builder.go               // Построение цепочек фильтров
│   └── validator.go             // Проверка конфликтов и валидация
├── routing/
│   ├── builder.go               // Построение конфигурации маршрутизации
│   └── virtual_host.go          // Построение виртуальных хостов
├── logging/
│   └── builder.go               // Построение конфигурации доступа
├── tls/
│   ├── builder.go               // Определение типа TLS
│   └── domain_mapping.go        // Связь доменов и секретов
├── utils/                       // Уже существующий пакет
│   ├── pools.go                 // Уже существует
│   ├── lru.go                   // Уже существует
│   ├── metrics.go               // Уже существует
│   └── helpers.go               // Новый файл для вспомогательных функций
├── clusters/                    // Уже существующий пакет
├── filters/                     // Уже существующий пакет
├── routes/                      // Уже существующий пакет
└── secrets/                     // Уже существующий пакет
```

## Интерфейсы компонентов

Для улучшения тестируемости и снижения связанности будут определены следующие интерфейсы:

```go
// HTTPFilterBuilder отвечает за построение HTTP фильтров
type HTTPFilterBuilder interface {
    BuildHTTPFilters(vs *v1alpha1.VirtualService) ([]*hcmv3.HttpFilter, error)
    BuildRBACFilter(vs *v1alpha1.VirtualService) (*rbacFilter.RBAC, error)
}

// FilterChainBuilder отвечает за построение цепочек фильтров
type FilterChainBuilder interface {
    BuildFilterChains(params *FilterChainsParams) ([]*listenerv3.FilterChain, error)
    BuildFilterChainParams(vs *v1alpha1.VirtualService, nn helpers.NamespacedName, 
                        httpFilters []*hcmv3.HttpFilter, listenerIsTLS bool, 
                        virtualHost *routev3.VirtualHost) (*FilterChainsParams, error)
    CheckFilterChainsConflicts(vs *v1alpha1.VirtualService) error
}

// RoutingBuilder отвечает за построение конфигурации маршрутизации
type RoutingBuilder interface {
    BuildRouteConfiguration(vs *v1alpha1.VirtualService, xdsListener *listenerv3.Listener, 
                          nn helpers.NamespacedName) (*routev3.VirtualHost, *routev3.RouteConfiguration, error)
    BuildVirtualHost(vs *v1alpha1.VirtualService, nn helpers.NamespacedName) (*routev3.VirtualHost, error)
}

// AccessLogBuilder отвечает за построение конфигурации доступа
type AccessLogBuilder interface {
    BuildAccessLogConfigs(vs *v1alpha1.VirtualService) ([]*accesslogv3.AccessLog, error)
}

// TLSBuilder отвечает за построение TLS конфигурации
type TLSBuilder interface {
    GetTLSType(vsTLSConfig *v1alpha1.TlsConfig) (string, error)
    GetSecretNameToDomains(vs *v1alpha1.VirtualService, domains []string) (map[helpers.NamespacedName][]string, error)
}

// ClusterExtractor отвечает за извлечение кластеров
type ClusterExtractor interface {
    ExtractClustersFromFilterChains(filterChains []*listenerv3.FilterChain) ([]*cluster.Cluster, error)
    ExtractClustersFromVirtualHost(virtualHost *routev3.VirtualHost) ([]*cluster.Cluster, error)
    ExtractClustersFromHTTPFilters(httpFilters []*hcmv3.HttpFilter) ([]*cluster.Cluster, error)
}
```

## Основной ResourceBuilder

Основной класс `ResourceBuilder` будет содержать ссылки на все необходимые интерфейсы:

```go
type ResourceBuilder struct {
    store            *store.Store
    httpFilterBuilder HTTPFilterBuilder
    filterChainBuilder FilterChainBuilder
    routingBuilder   RoutingBuilder
    accessLogBuilder AccessLogBuilder
    tlsBuilder       TLSBuilder
    clustersBuilder  clusters.Builder
    clusterExtractor ClusterExtractor
    secretsBuilder   secrets.Builder
}
```

## Стратегия тестирования

1. **Модульные тесты** для каждого компонента:
   - Тесты для каждого интерфейса и его реализации
   - Моки для зависимостей с использованием интерфейсов

2. **Интеграционные тесты** для проверки взаимодействия компонентов:
   - Тесты основного `ResourceBuilder` с реальными реализациями компонентов
   - Проверка корректности собранных ресурсов

3. **Тесты эквивалентности** для сравнения с оригинальной реализацией:
   - Запуск одинаковых тестовых сценариев на старой и новой реализациях
   - Сравнение результатов для подтверждения эквивалентности

## План реализации

1. **Подготовка** (1 день):
   - Создание структуры директорий и пакетов
   - Определение интерфейсов
   - Разработка стратегии миграции

2. **Миграция компонентов** (3-4 дня):
   - Последовательный перенос каждого компонента в свой пакет
   - Реализация интерфейсов
   - Написание тестов для каждого компонента

3. **Рефакторинг основного билдера** (1-2 дня):
   - Обновление `ResourceBuilder` для использования компонентов через интерфейсы
   - Сокращение и упрощение основного файла builder.go

4. **Тестирование и отладка** (2 дня):
   - Запуск всех тестов
   - Проверка эквивалентности результатов
   - Исправление обнаруженных проблем

5. **Проверка производительности** (1 день):
   - Запуск бенчмарков
   - Сравнение производительности до и после рефакторинга
   - Оптимизация при необходимости

## Ожидаемые результаты

1. **Улучшение модульности**:
   - Меньшие по размеру файлы с четкой ответственностью
   - Снижение когнитивной нагрузки при работе с кодом

2. **Улучшение тестируемости**:
   - Возможность тестировать компоненты изолированно
   - Лучшее покрытие тестами

3. **Улучшение поддерживаемости**:
   - Легче понимать и модифицировать компоненты
   - Явная структура зависимостей

4. **Снижение сложности**:
   - Уменьшение цикломатической сложности функций
   - Лучшая структура кода

5. **Сохранение производительности**:
   - Отсутствие регрессии в производительности
   - Потенциал для дальнейших оптимизаций

## Критерии успеха

1. Все тесты проходят успешно
2. Бенчмарки показывают производительность не хуже исходной
3. Размер файлов не превышает 300 строк
4. Цикломатическая сложность функций снижена
5. Покрытие кода тестами увеличено до >80%