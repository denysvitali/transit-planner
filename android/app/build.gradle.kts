import java.util.Properties

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val keystoreProperties = Properties().apply {
    val keystorePropertiesFile = rootProject.file("key.properties")
    if (keystorePropertiesFile.exists()) {
        keystorePropertiesFile.inputStream().use { load(it) }
    }
}

fun envOrProp(envName: String, propName: String): String? =
    (System.getenv(envName)?.takeIf { it.isNotBlank() }
        ?: keystoreProperties.getProperty(propName))?.takeIf { it.isNotBlank() }

val keystorePath: String? = envOrProp("KEYSTORE_PATH", "storeFile")
val keystoreStorePassword: String? = envOrProp("KEYSTORE_STORE_PASSWORD", "storePassword")
val keystoreKeyPassword: String? = envOrProp("KEYSTORE_KEY_PASSWORD", "keyPassword")
val keystoreKeyAlias: String? = envOrProp("KEYSTORE_KEY_ALIAS", "keyAlias")

android {
    namespace = "it.denv.transit_planner"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    defaultConfig {
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "it.denv.transit_planner"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        create("release") {
            keystorePath?.let { storeFile = file(it) }
            storePassword = keystoreStorePassword
            keyAlias = keystoreKeyAlias
            keyPassword = keystoreKeyPassword
        }
    }

    buildTypes {
        release {
            // Use the release signing config when a keystore is provided via
            // env vars / key.properties; otherwise fall back to debug keys so
            // local `flutter run --release` still works.
            signingConfig = if (keystorePath != null) {
                signingConfigs.getByName("release")
            } else {
                signingConfigs.getByName("debug")
            }
        }
    }
}

flutter {
    source = "../.."
}
